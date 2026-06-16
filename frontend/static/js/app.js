/**
 * League Simulation — Vue 3 Application
 *
 * Separation of Concerns:
 *   - api.js    → HTTP calls (imported via <script> before this file)
 *   - styles.css → Presentation
 *   - app.js    → State management, business logic, UI orchestration
 */
const { createApp, ref, computed, onMounted, nextTick } = Vue;

createApp({
  setup() {
    // ─── Reactive State ───
    const blankTeam = () => ({
      name: '',
      strength: 70,
      attack_rating: 70,
      defense_rating: 70,
      form_rating: 70,
    });

    const teams = ref([]);
    const matches = ref([]);
    const standings = ref([]);
    const predictions = ref([]);
    const predictionMeta = ref({ current_week: 0, available_after_week: 4, simulations: 0 });

    const historicalStats = ref([]);
    const historicalMatches = ref([]);
    const newTeam = ref(blankTeam());
    const bulkTeamText = ref('Bosphorus United,74,78,70,76\nAnka FC,69,72,67,71\nGalata Rovers,81,84,79,80\nModa Athletic,66,68,65,69');
    const bulkReplace = ref(false);

    // Chat
    const chatMessages = ref([
      { role: 'assistant', content: 'Ask me about this league: title favorite, current table, predictions, next fixtures, form archive, or the simulation model.' },
    ]);
    const chatInput = ref('');
    const chatLoading = ref(false);
    const chatContainer = ref(null);
    const chatInputEl = ref(null);
    const chatOpen = ref(false);
    const chatVisible = ref(false);
    const chatClosing = ref(false);
    const unreadCount = ref(0);

    // Modals
    const historicalOpen = ref(false);
    const historicalSearch = ref('');
    const historicalSeason = ref('');
    const editOpen = ref(false);
    const editMatch = ref(null);
    const editHomeScore = ref(0);
    const editAwayScore = ref(0);

    // UI state
    const viewWeek = ref(1);
    const loading = ref(false);
    const simulating = ref(false);
    const animateScores = ref(false);
    const animateTable = ref(false);
    const statusMsg = ref('');
    const statusType = ref('');

    // ─── Computed Properties ───
    const teamMap = computed(() => {
      const map = {};
      teams.value.forEach(t => map[t.id] = t.name);
      return map;
    });

    const teamName = (id) => teamMap.value[id] || 'Unknown';

    const fixturesExist = computed(() => matches.value.length > 0);
    const rosterLocked = computed(() => fixturesExist.value);
    const totalWeeks = computed(() => {
      if (!matches.value.length) return 0;
      return Math.max(...matches.value.map(m => m.week));
    });

    const allPlayed = computed(() => {
      if (!matches.value.length) return true;
      return matches.value.every(m => m.played);
    });

    const currentWeekMatches = computed(() =>
      matches.value.filter(m => m.week === viewWeek.value)
    );

    const predictionsVisible = computed(() =>
      predictionMeta.value.current_week >= predictionMeta.value.available_after_week
    );

    const predictionEmptyMessage = computed(() => {
      if (!fixturesExist.value) return 'Build schedule first';
      return `Predictions become visible after week ${predictionMeta.value.available_after_week}`;
    });

    const rosterMessage = computed(() => {
      if (rosterLocked.value) return 'Roster locked';
      if (teams.value.length < 4) return `${teams.value.length}/4 teams`;
      if (teams.value.length % 2 !== 0) return 'Odd team count';
      return 'Ready';
    });

    const rosterMessageType = computed(() => {
      if (rosterLocked.value) return 'locked';
      if (teams.value.length >= 4 && teams.value.length % 2 === 0) return 'ready';
      return 'warning';
    });

    const historicalSeasons = computed(() =>
      [...new Set(historicalMatches.value.map(m => m.season))].sort()
    );

    const filteredHistoricalMatches = computed(() => {
      const search = historicalSearch.value.trim().toLowerCase();
      return historicalMatches.value.filter(m => {
        const matchesSeason = !historicalSeason.value || m.season === historicalSeason.value;
        const matchesSearch = !search ||
          m.season.toLowerCase().includes(search) ||
          m.home_team.toLowerCase().includes(search) ||
          m.away_team.toLowerCase().includes(search);
        return matchesSeason && matchesSearch;
      });
    });

    const historicalStatsWithNames = computed(() =>
      historicalStats.value.map(s => ({ ...s, team_name: teamName(s.team_id) }))
    );

    // ─── UI Helpers ───
    const showStatus = (msg, type = 'success') => {
      statusMsg.value = msg;
      statusType.value = type;
      setTimeout(() => { statusMsg.value = ''; }, 3000);
    };

    const triggerAnimations = () => {
      animateScores.value = false;
      animateTable.value = false;
      nextTick(() => {
        animateScores.value = true;
        animateTable.value = true;
        setTimeout(() => { animateScores.value = false; animateTable.value = false; }, 2000);
      });
    };

    const scrollChat = () => {
      nextTick(() => {
        if (chatContainer.value) {
          chatContainer.value.scrollTop = chatContainer.value.scrollHeight;
        }
      });
    };

    const asRating = (value, fallback = 70) => {
      const rating = Number(value);
      if (!Number.isFinite(rating)) return fallback;
      return Math.max(1, Math.min(100, Math.round(rating)));
    };

    const teamPayload = (team) => {
      const name = String(team.name || '').trim();
      const strength = asRating(team.strength, 70);
      return {
        name,
        strength,
        attack_rating: asRating(team.attack_rating, strength),
        defense_rating: asRating(team.defense_rating, strength),
        form_rating: asRating(team.form_rating, strength),
      };
    };

    const parseTeamImport = (text) => {
      return text
        .split(/\r?\n/)
        .map(line => line.trim())
        .filter(Boolean)
        .map((line, index) => {
          const delimiter = line.includes('\t') ? '\t' : line.includes(';') ? ';' : ',';
          const cells = line.split(delimiter).map(cell => cell.trim());
          if (index === 0 && cells[0]?.toLowerCase() === 'name') return null;
          const strength = asRating(cells[1], 70);
          return teamPayload({
            name: cells[0],
            strength,
            attack_rating: cells[2] || strength,
            defense_rating: cells[3] || strength,
            form_rating: cells[4] || strength,
          });
        })
        .filter(team => team && team.name);
    };

    // ─── Data Fetching (uses LeagueAPI from api.js) ───
    const fetchAll = async () => {
      try {
        const [tRes, mRes, sRes] = await Promise.all([
          LeagueAPI.getTeams(),
          LeagueAPI.getMatches(),
          LeagueAPI.getStandings(),
        ]);
        teams.value = tRes.data?.teams || [];
        matches.value = mRes.data?.matches || [];
        standings.value = sRes.data?.standings || [];

        if (matches.value.length) {
          const pRes = await LeagueAPI.getPredictions();
          if (pRes.success) {
            predictions.value = pRes.data?.predictions || [];
            predictionMeta.value = {
              current_week: pRes.data?.current_week || 0,
              available_after_week: pRes.data?.available_after_week || 4,
              simulations: pRes.data?.simulations || 0,
            };
          } else {
            predictions.value = [];
          }
        } else {
          predictions.value = [];
          predictionMeta.value = { current_week: 0, available_after_week: 4, simulations: 0 };
        }
      } catch (e) {
        showStatus('Failed to load data', 'error');
      }
    };

    const fetchHistoricalStats = async () => {
      try {
        const res = await LeagueAPI.getHistoricalStats();
        if (res.success) historicalStats.value = res.data?.stats || [];
      } catch (e) {
        showStatus('Failed to load historical stats', 'error');
      }
    };

    const fetchHistoricalMatches = async () => {
      try {
        const res = await LeagueAPI.getHistoricalMatches();
        if (res.success) {
          historicalMatches.value = res.data?.matches || [];
        } else {
          showStatus(res.message, 'error');
        }
      } catch (e) {
        showStatus('Failed to load historical data', 'error');
      }
    };

    // ─── Team Studio ───
    const saveTeam = async () => {
      if (rosterLocked.value) return;

      const payload = teamPayload(newTeam.value);
      if (!payload.name) {
        showStatus('Team name is required', 'error');
        return;
      }

      loading.value = true;
      try {
        const res = await LeagueAPI.saveTeam(payload);
        if (res.success) {
          newTeam.value = blankTeam();
          await fetchAll();
          await fetchHistoricalStats();
          showStatus('Team saved');
        } else {
          showStatus(res.message, 'error');
        }
      } catch (e) {
        showStatus('Failed to save team', 'error');
      }
      loading.value = false;
    };

    const importTeams = async () => {
      if (rosterLocked.value) return;

      const imported = parseTeamImport(bulkTeamText.value);
      if (!imported.length) {
        showStatus('Add at least one team to import', 'error');
        return;
      }

      loading.value = true;
      try {
        const res = await LeagueAPI.importTeams(imported, bulkReplace.value);
        if (res.success) {
          await fetchAll();
          await fetchHistoricalStats();
          if (historicalMatches.value.length) await fetchHistoricalMatches();
          showStatus(`${res.data?.teams?.length || imported.length} teams imported`);
        } else {
          showStatus(res.message, 'error');
        }
      } catch (e) {
        showStatus('Failed to import teams', 'error');
      }
      loading.value = false;
    };

    const deleteTeam = async (team) => {
      if (rosterLocked.value || !team?.id) return;
      if (!window.confirm(`Delete ${team.name}?`)) return;

      loading.value = true;
      try {
        const res = await LeagueAPI.deleteTeam(team.id);
        if (res.success) {
          await fetchAll();
          await fetchHistoricalStats();
          showStatus('Team deleted');
        } else {
          showStatus(res.message, 'error');
        }
      } catch (e) {
        showStatus('Failed to delete team', 'error');
      }
      loading.value = false;
    };

    // ─── Chat ───
    const sendChatMessage = async () => {
      const message = chatInput.value.trim();
      if (!message || chatLoading.value) return;

      chatMessages.value.push({ role: 'user', content: message });
      chatInput.value = '';
      chatLoading.value = true;
      scrollChat();

      try {
        const history = chatMessages.value
          .filter(m => m.role === 'user' || m.role === 'assistant')
          .slice(-8)
          .map(m => ({ role: m.role, content: m.content }));

        const res = await LeagueAPI.chat(message, history);

        if (res.success) {
          chatMessages.value.push({
            role: 'assistant',
            content: res.data?.answer || 'I could not produce an answer.',
            source: res.data?.source,
            model: res.data?.model,
            tool_calls: res.data?.tool_calls || [],
          });
        } else {
          chatMessages.value.push({ role: 'assistant', content: res.message || 'Agent request failed.' });
        }
      } catch (e) {
        chatMessages.value.push({ role: 'assistant', content: 'Agent request failed.' });
      }

      chatLoading.value = false;
      if (!chatOpen.value) unreadCount.value++;
      scrollChat();
    };

    const openChat = () => {
      chatOpen.value = true;
      chatVisible.value = true;
      chatClosing.value = false;
      unreadCount.value = 0;
      nextTick(() => {
        scrollChat();
        if (chatInputEl.value) chatInputEl.value.focus();
      });
    };

    const closeChat = () => {
      chatClosing.value = true;
      chatOpen.value = false;
      setTimeout(() => {
        chatVisible.value = false;
        chatClosing.value = false;
      }, 260);
    };

    // ─── Modals ───
    const openHistoricalModal = async () => {
      historicalOpen.value = true;
      if (!historicalMatches.value.length) {
        await fetchHistoricalMatches();
      }
    };

    const openEditMatch = (match) => {
      editMatch.value = match;
      editHomeScore.value = match.home_score ?? 0;
      editAwayScore.value = match.away_score ?? 0;
      editOpen.value = true;
    };

    const saveMatchEdit = async () => {
      if (!editMatch.value) return;
      loading.value = true;
      try {
        const res = await LeagueAPI.updateMatch(editMatch.value.id, Number(editHomeScore.value), Number(editAwayScore.value));
        if (res.success) {
          editOpen.value = false;
          await fetchAll();
          triggerAnimations();
          showStatus('Match updated!');
        } else {
          showStatus(res.message, 'error');
        }
      } catch (e) {
        showStatus('Failed to update match', 'error');
      }
      loading.value = false;
    };

    // ─── Game Actions ───
    const generateFixtures = async () => {
      loading.value = true;
      try {
        const res = await LeagueAPI.generateFixtures();
        if (res.success) {
          await fetchAll();
          viewWeek.value = 1;
          showStatus('Schedule built');
        } else {
          showStatus(res.message, 'error');
        }
      } catch (e) {
        showStatus('Failed to build schedule', 'error');
      }
      loading.value = false;
    };

    const playNextWeek = async () => {
      loading.value = true;
      simulating.value = true;
      try {
        const res = await LeagueAPI.playNextWeek();
        simulating.value = false;
        if (res.success) {
          await fetchAll();
          viewWeek.value = res.data.week;
          triggerAnimations();
          showStatus(`Week ${res.data.week} played!`);
        } else {
          showStatus(res.message, 'error');
        }
      } catch (e) {
        simulating.value = false;
        showStatus('Failed to play week', 'error');
      }
      loading.value = false;
    };

    const playAll = async () => {
      loading.value = true;
      simulating.value = true;
      try {
        const res = await LeagueAPI.playAll();
        simulating.value = false;
        if (!res.success) {
          showStatus(res.message, 'error');
          loading.value = false;
          return;
        }
        await fetchAll();
        if (res.data?.weeks?.length) {
          viewWeek.value = res.data.weeks[res.data.weeks.length - 1].week;
        }
        triggerAnimations();
        showStatus('All weeks played! Season complete. 🏆');
      } catch (e) {
        simulating.value = false;
        showStatus('Failed during play all', 'error');
      }
      loading.value = false;
    };

    const resetLeague = async () => {
      loading.value = true;
      try {
        const res = await LeagueAPI.reset();
        if (res.success) {
          await fetchAll();
          viewWeek.value = 1;
          showStatus('Season cleared');
        } else {
          showStatus(res.message, 'error');
        }
      } catch (e) {
        showStatus('Failed to reset', 'error');
      }
      loading.value = false;
    };

    // ─── Lifecycle ───
    onMounted(() => {
      fetchAll();
      fetchHistoricalStats();
    });

    // ─── Template API ───
    return {
      teams, matches, standings, predictions, predictionMeta,
      newTeam, bulkTeamText, bulkReplace,
      chatMessages, chatInput, chatLoading, chatContainer, chatInputEl,
      chatOpen, chatVisible, chatClosing, unreadCount,
      historicalStats, viewWeek, loading, simulating, animateScores, animateTable,
      historicalOpen, historicalMatches, historicalSearch, historicalSeason,
      editOpen, editMatch, editHomeScore, editAwayScore,
      statusMsg, statusType,
      fixturesExist, rosterLocked, rosterMessage, rosterMessageType,
      totalWeeks, allPlayed, currentWeekMatches, predictionsVisible, predictionEmptyMessage,
      historicalSeasons, filteredHistoricalMatches, historicalStatsWithNames,
      teamName, generateFixtures, playNextWeek, playAll, resetLeague, openHistoricalModal,
      saveTeam, importTeams, deleteTeam, openEditMatch, saveMatchEdit, sendChatMessage, openChat, closeChat,
    };
  }
}).mount('#app');
