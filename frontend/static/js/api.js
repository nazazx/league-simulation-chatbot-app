/**
 * LeagueAPI — Centralized API client for all backend calls.
 * Single Responsibility: If the API URL or headers change, only this file is updated.
 * DRY: All fetch calls are funnelled through get() and post() helpers.
 */
const API_BASE = '/api/v1';

const LeagueAPI = {
  // --- Private helpers ---
  async _get(path) {
    const res = await fetch(`${API_BASE}${path}`);
    return res.json();
  },

  async _post(path, body = null) {
    const options = { method: 'POST' };
    if (body) {
      options.headers = { 'Content-Type': 'application/json' };
      options.body = JSON.stringify(body);
    }
    const res = await fetch(`${API_BASE}${path}`, options);
    return res.json();
  },

  async _put(path, body) {
    const res = await fetch(`${API_BASE}${path}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    return res.json();
  },

  // --- Public API ---
  getTeams()              { return this._get('/teams'); },
  getMatches()            { return this._get('/matches'); },
  getStandings()          { return this._get('/standings'); },
  getPredictions()        { return this._get('/predictions'); },
  getHistoricalStats()    { return this._get('/historical-stats'); },
  getHistoricalMatches()  { return this._get('/historical-matches'); },

  saveTeam(team)           { return this._post('/teams', team); },
  importTeams(teams, replace = false) {
    return this._post('/teams/import', { teams, replace });
  },
  deleteTeam(id) {
    return fetch(`${API_BASE}/teams/${id}`, { method: 'DELETE' }).then(res => res.json());
  },

  generateFixtures()      { return this._post('/fixtures/generate'); },
  playNextWeek()          { return this._post('/play/week'); },
  playAll()               { return this._post('/play/all'); },
  reset()                 { return this._post('/reset'); },

  updateMatch(id, homeScore, awayScore) {
    return this._put(`/matches/${id}`, { home_score: homeScore, away_score: awayScore });
  },

  chat(message, history) {
    return this._post('/agent/chat', { message, history });
  },
};
