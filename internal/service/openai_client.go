package service

// LEARNING NOTE: Bu dosya OpenAI ile HTTP üzerinden konuşan düşük seviye client kodunu içerir.
// Insight/chat servisinin "OpenAI'ye request at, response'u Go struct'ına çevir" ihtiyacını karşılar.
// Burada business logic yoktur; sadece external API entegrasyonu ve JSON parse işlemleri vardır.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const openAIResponsesURL = "https://api.openai.com/v1/responses"

// --- OpenAI Responses API types ---

type openAIResponsesRequest struct {
	// LEARNING NOTE: Bu struct OpenAI Responses API'ye gönderilen JSON body'nin Go karşılığıdır.
	// Struct field'larının sonundaki `json:"model"` gibi tag'ler, Go field adının JSON'da hangi isimle gideceğini söyler.
	Model              string       `json:"model"`
	Instructions       string       `json:"instructions,omitempty"`
	Input              interface{}  `json:"input"`
	Tools              []openAITool `json:"tools,omitempty"`
	PreviousResponseID string       `json:"previous_response_id,omitempty"`
	MaxOutputTokens    int          `json:"max_output_tokens"`
}

type openAITool struct {
	// LEARNING NOTE: Tool tanımı, modele hangi backend fonksiyonlarını çağırabileceğini anlatır.
	// `Parameters` alanı JSON schema benzeri bir map'tir; model hangi argümanları gönderebileceğini buradan öğrenir.
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Strict      bool                   `json:"strict"`
}

type openAIFunctionCallOutput struct {
	// LEARNING NOTE: Model bir tool çağırınca backend o tool'u çalıştırır ve sonucu bu struct ile OpenAI'ye geri yollar.
	// CallID, modelin istediği tool çağrısı ile backend'in döndürdüğü sonucu eşleştirmek için kullanılır.
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

type openAIResponsesResponse struct {
	// LEARNING NOTE: OpenAI cevabı iki farklı şekilde gelebilir: final text veya function_call.
	// Bu yüzden response struct'ı hem OutputText'i hem de Output listesi içindeki tool çağrılarını tutar.
	ID         string `json:"id"`
	OutputText string `json:"output_text"`
	Output     []struct {
		Type      string `json:"type"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
		CallID    string `json:"call_id"`
		Content   []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

type openAIFunctionCall struct {
	// LEARNING NOTE: Bu küçük internal struct, OpenAI response içindeki function_call verisini daha kolay taşımak için kullanılır.
	Name      string
	Arguments string
	CallID    string
}

// firstText returns the first non-empty text content from the response output.
func (r openAIResponsesResponse) firstText() string {
	// LEARNING NOTE: `(r openAIResponsesResponse)` method receiver'dır. Bu fonksiyon openAIResponsesResponse tipine aitmiş gibi çağrılır.
	// Bazı OpenAI cevaplarında OutputText boş olabilir; o durumda output.content içindeki ilk text aranır.
	for _, output := range r.Output {
		for _, content := range output.Content {
			if strings.TrimSpace(content.Text) != "" {
				return content.Text
			}
		}
	}
	return ""
}

// functionCalls extracts all function_call items from the response.
func (r openAIResponsesResponse) functionCalls() []openAIFunctionCall {
	// LEARNING NOTE: Model tool çağırmak isterse response output içinde function_call item'ları döner.
	// Bu fonksiyon sadece Type == "function_call" olan item'ları süzer ve chat loop'una verir.
	var calls []openAIFunctionCall
	for _, output := range r.Output {
		if output.Type != "function_call" {
			continue
		}
		calls = append(calls, openAIFunctionCall{
			Name:      output.Name,
			Arguments: output.Arguments,
			CallID:    output.CallID,
		})
	}
	return calls
}

// --- HTTP Client ---

// createOpenAIResponse sends a request to the OpenAI Responses API and returns the parsed response.
func (s *insightService) createOpenAIResponse(ctx context.Context, requestBody openAIResponsesRequest) (*openAIResponsesResponse, error) {
	// LEARNING NOTE: Bu external API çağrısıdır; hata olursa üst katmanda fallback cevap üretilir.
	// `s *insightService` receiver olduğu için bu metot insightService'in openAIKey ve httpClient alanlarını kullanabilir.
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIResponsesURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	// LEARNING NOTE: Authorization header API key'i taşır. Content-Type ise gönderdiğimiz body'nin JSON olduğunu söyler.
	req.Header.Set("Authorization", "Bearer "+s.openAIKey)
	req.Header.Set("Content-Type", "application/json")

	// LEARNING NOTE: HTTP request burada gerçekten gönderilir. Network/API hatası olursa err dolu gelir.
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	// LEARNING NOTE: defer resp.Body.Close(), response body okunduktan sonra bağlantı kaynağını serbest bırakır.
	defer resp.Body.Close()

	// LEARNING NOTE: io.ReadAll response body'yi byte slice olarak okur; sonra JSON parse etmek için kullanılır.
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// LEARNING NOTE: 2xx dışı HTTP status'lar API hatası kabul edilir. Hata üst katmana dönünce fallback devreye girebilir.
		return nil, fmt.Errorf("openai responses api returned status %d: %s", resp.StatusCode, truncateForLog(string(responseBody), 600))
	}

	var openAIResp openAIResponsesResponse
	// LEARNING NOTE: json.Unmarshal, JSON byte verisini Go struct'ına çevirir.
	if err := json.Unmarshal(responseBody, &openAIResp); err != nil {
		return nil, err
	}
	return &openAIResp, nil
}

// mustJSON marshals a value to JSON, returning "{}" on error.
func mustJSON(value interface{}) string {
	// LEARNING NOTE: Bu yardımcı fonksiyon tool output'larını JSON string'e çevirir. Hata olursa boş object döndürerek akışı kırmaz.
	b, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func truncateForLog(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}
