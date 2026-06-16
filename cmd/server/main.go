package main

// LEARNING NOTE: Bu dosya uygulamanın "composition root" kısmıdır. Yani config, database,
// repository, service, handler ve router burada birbirine bağlanır. Request burada işlenmez;
// sadece backend'in çalışması için gerekli parçalar kurulup HTTP server başlatılır.

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/nazazx/league-simulation/internal/config"
	"github.com/nazazx/league-simulation/internal/database"
	"github.com/nazazx/league-simulation/internal/handler"
	"github.com/nazazx/league-simulation/internal/repository"
	"github.com/nazazx/league-simulation/internal/router"
	"github.com/nazazx/league-simulation/internal/service"
)

func main() {
	// LEARNING NOTE: Go programlarında çalıştırılabilir uygulamanın ilk girilen fonksiyonu main()'dir.
	// Bu proje bir backend olduğu için main() içinde server ayağa kaldırılır.
	// 1. Load configuration
	cfg := config.Load()

	// 2. Connect to database
	// LEARNING NOTE: `:=` Go'da kısa değişken tanımlama syntax'ıdır. Burada hem db hem err aynı anda oluşturulur.
	// Fonksiyonlar Go'da birden fazla değer döndürebilir; NewPostgresDB hem bağlantı hem hata döndürüyor.
	db, err := database.NewPostgresDB(cfg.DSN())
	if err != nil {
		// LEARNING NOTE: Go'da exception yoktur; hata değeri elle kontrol edilir. Kritik hatada log.Fatalf uygulamayı durdurur.
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// LEARNING NOTE: defer, fonksiyon bitmeden hemen önce çalışır. Server kapanırken DB bağlantısı düzgün kapatılır.
	defer db.Close()

	// 3. Initialize repositories
	// LEARNING NOTE: Repository katmanı veritabanı işlemlerini saklar. Service katmanı "takımları getir" der,
	// SQL'in nasıl yazıldığını bilmez. Bu ayrım backend mimarisinde sorumlulukları temiz ayırır.
	teamRepo := repository.NewTeamRepository(db)
	matchRepo := repository.NewMatchRepository(db)

	// 4. Initialize services
	// LEARNING NOTE: Dependency injection burada görülür. Örneğin MatchService kendi içinde DB yaratmaz;
	// ihtiyaç duyduğu teamRepo, matchRepo ve simulator dışarıdan verilir. Bu test etmeyi ve değiştirmeyi kolaylaştırır.
	sim := service.NewSimulator()
	fixtureSvc := service.NewFixtureService(teamRepo, matchRepo)
	matchSvc := service.NewMatchService(teamRepo, matchRepo, sim)
	standingSvc := service.NewStandingService(matchRepo)
	predictionSvc := service.NewPredictionService(teamRepo, matchRepo, sim)
	insightSvc := service.NewInsightService(matchRepo, predictionSvc, cfg.OpenAIKey, cfg.OpenAIModel)

	// 5. Initialize handler
	// LEARNING NOTE: Handler HTTP isteğini karşılayan katmandır. Request body, query param ve status code gibi
	// HTTP detayları burada kalır; fikstür üretme veya prediction gibi iş kuralları service katmanındadır.
	h := handler.NewHandler(db, teamRepo, matchRepo, fixtureSvc, matchSvc, standingSvc, predictionSvc, insightSvc)

	// 6. Setup router
	// LEARNING NOTE: Router, "GET /api/v1/teams gelirse hangi fonksiyon çalışacak?" sorusunun cevabını tutar.
	r := router.Setup(h)

	// 7. Create HTTP server with graceful shutdown
	// LEARNING NOTE: &http.Server{...} bir struct literal'ının adresini alır. Handler alanına Gin router verilir,
	// böylece gelen HTTP request'ler Gin üzerinden ilgili endpoint'e gider.
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Listen for shutdown signals (SIGINT, SIGTERM)
	// LEARNING NOTE: signal.NotifyContext, Ctrl+C veya container stop gibi sinyalleri context üzerinden yakalar.
	// Bu sayede server aniden kesilmek yerine aktif request'leri bitirmeye çalışır.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start server in a goroutine
	// LEARNING NOTE: `go func() { ... }()` syntax'ı anonim fonksiyonu yeni goroutine'de çalıştırır.
	// Server dinleme işi bloklayıcı olduğu için arkaya alınır; main goroutine aşağıda shutdown sinyali bekler.
	go func() {
		log.Printf("Starting server on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Block until shutdown signal is received
	<-ctx.Done()
	log.Println("Shutdown signal received, draining active requests...")

	// Give active requests up to 10 seconds to complete
	// LEARNING NOTE: Graceful shutdown, aktif request'lere bitmesi için süre tanır. 10 saniye dolarsa server zorla kapanır.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
