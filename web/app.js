// Untis Go - Modern PC Desktop Remake v1.0.0
// Material 3 + Desktop Design Tokens, Untis Orange, Zero-Lag SQLite Cache

(function () {
  'use strict';

  // 1. Session Token & Authenticated Fetch
  const urlParams = new URLSearchParams(window.location.search);
  const sessionToken = urlParams.get('token') || '';

  async function authFetch(url, options = {}) {
    options.headers = options.headers || {};
    if (sessionToken) {
      options.headers['X-Session-Token'] = sessionToken;
    }
    const separator = url.includes('?') ? '&' : '?';
    const finalUrl = sessionToken ? `${url}${separator}token=${encodeURIComponent(sessionToken)}` : url;
    return fetch(finalUrl, options);
  }

  // 2. Application State
  const state = {
    needsOnboarding: false,
    classes: [],
    filteredClasses: [],
    selectedClassId: 0,
    selectedClassName: '',
    currentDate: new Date(),
    currentView: 'day', // 'day' | 'week'
    theme: 'dark',
    schoolName: '',
    serverUrl: '',
    userDisplayName: 'Online',
    profiles: [],
    activeProfileId: '',
    lessons: [],
    loading: false,
    classSearchTerm: '',
    onboardingSelectedSchool: null
  };

  // 3. DOM Elements
  const el = {
    // Top App Bar
    schoolSubtitle: document.getElementById('schoolSubtitle'),
    userDisplayName: document.getElementById('userDisplayName'),
    userAvatarBadge: document.getElementById('userAvatarBadge'),
    profileBadgeBtn: document.getElementById('profileBadgeBtn'),
    profileDropdownWrapper: document.querySelector('.profile-dropdown-wrapper'),
    profileDropdownMenu: document.getElementById('profileDropdownMenu'),
    profileQuickList: document.getElementById('profileQuickList'),
    addNewProfileQuickBtn: document.getElementById('addNewProfileQuickBtn'),
    classSelectorBtn: document.getElementById('classSelectorBtn'),
    classDropdownWrapper: document.querySelector('.class-dropdown-wrapper'),
    classDropdownMenu: document.getElementById('classDropdownMenu'),
    classSearchInput: document.getElementById('classSearchInput'),
    clearClassSearchBtn: document.getElementById('clearClassSearchBtn'),
    classCounterBadge: document.getElementById('classCounterBadge'),
    classListContainer: document.getElementById('classListContainer'),
    currentClassLabel: document.getElementById('currentClassLabel'),
    refreshBtn: document.getElementById('refreshBtn'),
    themeToggleBtn: document.getElementById('themeToggleBtn'),
    themeIcon: document.getElementById('themeIcon'),
    settingsBtn: document.getElementById('settingsBtn'),

    // Date & Navigation Control
    prevDateBtn: document.getElementById('prevDateBtn'),
    nextDateBtn: document.getElementById('nextDateBtn'),
    dateDisplayBtn: document.getElementById('dateDisplayBtn'),
    dateDisplayLabel: document.getElementById('dateDisplayLabel'),
    nativeDatePicker: document.getElementById('nativeDatePicker'),
    todayBtn: document.getElementById('todayBtn'),
    viewDayBtn: document.getElementById('viewDayBtn'),
    viewWeekBtn: document.getElementById('viewWeekBtn'),

    // Viewports
    mainContentViewport: document.getElementById('mainContentViewport'),
    loadingOverlay: document.getElementById('loadingOverlay'),
    loadingMessage: document.getElementById('loadingMessage'),
    onboardingScreen: document.getElementById('onboardingScreen'),
    onboardingSearchInput: document.getElementById('onboardingSearchInput'),
    onboardingSearchSpinner: document.getElementById('onboardingSearchSpinner'),
    onboardingSchoolResults: document.getElementById('onboardingSchoolResults'),
    onboardingCredentialsForm: document.getElementById('onboardingCredentialsForm'),
    onboardingSelectedSchoolName: document.getElementById('onboardingSelectedSchoolName'),
    onboardingSelectedSchoolDetails: document.getElementById('onboardingSelectedSchoolDetails'),
    onboardingChangeSchoolBtn: document.getElementById('onboardingChangeSchoolBtn'),
    onboardingLoginFormBox: document.getElementById('onboardingLoginFormBox'),
    onboardingUsernameInput: document.getElementById('onboardingUsernameInput'),
    onboardingPasswordInput: document.getElementById('onboardingPasswordInput'),
    toggleOnboardingPwdBtn: document.getElementById('toggleOnboardingPwdBtn'),
    onboardingErrorMsg: document.getElementById('onboardingErrorMsg'),
    onboardingConnectBtn: document.getElementById('onboardingConnectBtn'),

    dayViewContainer: document.getElementById('dayViewContainer'),
    weekViewContainer: document.getElementById('weekViewContainer'),
    dayLessonsList: document.getElementById('dayLessonsList'),
    liveTimelineMarker: document.getElementById('liveTimelineMarker'),
    liveTimelineTimeLabel: document.getElementById('liveTimelineTimeLabel'),
    weekGridHeader: document.getElementById('weekGridHeader'),
    weekGridBody: document.getElementById('weekGridBody'),
    emptyState: document.getElementById('emptyState'),
    emptyStateTitle: document.getElementById('emptyStateTitle'),
    emptyStateDesc: document.getElementById('emptyStateDesc'),
    jumpToNextWeekBtn: document.getElementById('jumpToNextWeekBtn'),

    // Lesson Detail Sheet
    sheetBackdrop: document.getElementById('sheetBackdrop'),
    lessonDetailSheet: document.getElementById('lessonDetailSheet'),
    sheetSubjectBadge: document.getElementById('sheetSubjectBadge'),
    sheetSubjectTitle: document.getElementById('sheetSubjectTitle'),
    sheetTimeTitle: document.getElementById('sheetTimeTitle'),
    sheetCloseBtn: document.getElementById('sheetCloseBtn'),
    sheetStatusRow: document.getElementById('sheetStatusRow'),
    sheetTeacherVal: document.getElementById('sheetTeacherVal'),
    sheetRoomVal: document.getElementById('sheetRoomVal'),
    sheetClassVal: document.getElementById('sheetClassVal'),
    sheetDateTimeVal: document.getElementById('sheetDateTimeVal'),
    sheetTeachingContentSection: document.getElementById('sheetTeachingContentSection'),
    sheetTeachingContentBox: document.getElementById('sheetTeachingContentBox'),
    sheetHomeworkSection: document.getElementById('sheetHomeworkSection'),
    sheetHomeworkBox: document.getElementById('sheetHomeworkBox'),
    sheetNotesSection: document.getElementById('sheetNotesSection'),
    sheetNotesBox: document.getElementById('sheetNotesBox'),

    // Settings Modal
    settingsModalBackdrop: document.getElementById('settingsModalBackdrop'),
    closeSettingsBtn: document.getElementById('closeSettingsBtn'),
    profileCardsList: document.getElementById('profileCardsList'),
    addNewProfileModalBtn: document.getElementById('addNewProfileModalBtn'),
    schoolSearchInput: document.getElementById('schoolSearchInput'),
    searchSchoolBtn: document.getElementById('searchSchoolBtn'),
    schoolResultsList: document.getElementById('schoolResultsList'),
    settingsForm: document.getElementById('settingsForm'),
    cfgServer: document.getElementById('cfgServer'),
    cfgSchool: document.getElementById('cfgSchool'),
    cfgUsername: document.getElementById('cfgUsername'),
    cfgPassword: document.getElementById('cfgPassword'),
    togglePasswordBtn: document.getElementById('togglePasswordBtn'),
    cfgProfileName: document.getElementById('cfgProfileName'),
    cfgTheme: document.getElementById('cfgTheme'),
    cfgDefaultView: document.getElementById('cfgDefaultView'),
    settingsStatusMsg: document.getElementById('settingsStatusMsg'),
    saveSettingsBtn: document.getElementById('saveSettingsBtn'),
    clearCacheBtn: document.getElementById('clearCacheBtn'),

    // Toast
    toastContainer: document.getElementById('toastContainer')
  };

  // Helper Functions
  function formatDateISO(d) {
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  function isSameDate(d1, d2) {
    return d1.getFullYear() === d2.getFullYear() &&
           d1.getMonth() === d2.getMonth() &&
           d1.getDate() === d2.getDate();
  }

  function getGermanWeekday(d) {
    const days = ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag'];
    return days[d.getDay()];
  }

  function getGermanMonth(d) {
    const months = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'];
    return months[d.getMonth()];
  }

  function showToast(msg) {
    const t = document.createElement('div');
    t.className = 'toast';
    t.textContent = msg;
    el.toastContainer.appendChild(t);
    setTimeout(() => {
      t.style.opacity = '0';
      t.style.transform = 'translateY(10px)';
      t.style.transition = 'all 0.25s ease';
      setTimeout(() => t.remove(), 250);
    }, 2800);
  }

  function getMondayOfWeek(d) {
    const date = new Date(d);
    const day = date.getDay();
    const diff = date.getDate() - day + (day === 0 ? -6 : 1);
    return new Date(date.setDate(diff));
  }

  function getFridayOfWeek(d) {
    const mon = getMondayOfWeek(d);
    const fri = new Date(mon);
    fri.setDate(mon.getDate() + 4);
    return fri;
  }

  function applyTheme(theme) {
    state.theme = theme;
    if (theme === 'system') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      document.documentElement.setAttribute('data-theme', prefersDark ? 'dark' : 'light');
    } else {
      document.documentElement.setAttribute('data-theme', theme);
    }
  }

  function getAvatarInitials(name, school) {
    if (name && name.trim()) {
      const parts = name.trim().split(/\s+/);
      if (parts.length >= 2) {
        return (parts[0][0] + parts[1][0]).toUpperCase();
      }
      return name.trim().substring(0, 2).toUpperCase();
    }
    if (school && school.trim()) {
      return school.trim().substring(0, 2).toUpperCase();
    }
    return 'UN';
  }

  // Application Initialization
  async function init() {
    setupEventListeners();
    await loadStatus();

    if (!state.needsOnboarding) {
      await loadProfiles();
      await loadClasses();
      updateDateDisplay();
      await loadTimetable();
    }

    // Start Live Time Line updates (every 30 seconds)
    setInterval(updateLiveTimeline, 30000);
  }

  // Load Status from SQLite Server
  async function loadStatus() {
    try {
      const res = await authFetch('/api/status');
      if (res.ok) {
        const data = await res.json();
        state.needsOnboarding = data.needsOnboarding === true;
        state.theme = data.theme || 'dark';
        state.currentView = data.defaultView || 'day';
        applyTheme(state.theme);

        if (state.needsOnboarding) {
          showOnboardingScreen(true);
          return;
        }

        showOnboardingScreen(false);
        state.activeProfileId = data.activeProfileId || '';
        state.selectedClassId = data.selectedClassId || 0;
        state.selectedClassName = data.selectedClassName || '';
        state.schoolName = data.school || '';
        state.serverUrl = data.server || '';

        state.userDisplayName = data.displayName || data.profileName || data.username || 'Online';
        el.userDisplayName.textContent = state.userDisplayName;
        el.userAvatarBadge.textContent = getAvatarInitials(state.userDisplayName, state.schoolName);
        el.schoolSubtitle.textContent = state.schoolName || 'WebUntis';

        updateViewToggle();

        // Prefill settings inputs
        el.cfgServer.value = state.serverUrl;
        el.cfgSchool.value = state.schoolName;
        el.cfgUsername.value = data.username || '';
        el.cfgProfileName.value = data.profileName || '';
        el.cfgTheme.value = state.theme;
        el.cfgDefaultView.value = state.currentView;
      }
    } catch (e) {
      console.warn('[Status] Fehler beim Laden:', e);
    }
  }

  // Onboarding Screen Management
  function showOnboardingScreen(show) {
    el.onboardingScreen.style.display = show ? 'flex' : 'none';
    if (show) {
      el.onboardingSearchInput.value = '';
      el.onboardingSchoolResults.innerHTML = '';
      el.onboardingCredentialsForm.style.display = 'none';
      setTimeout(() => el.onboardingSearchInput.focus(), 150);
    }
  }

  let searchTimeout = null;
  function handleOnboardingSearch(query) {
    clearTimeout(searchTimeout);
    const q = query.trim();
    if (!q) {
      el.onboardingSchoolResults.innerHTML = '';
      el.onboardingSearchSpinner.style.display = 'none';
      return;
    }

    el.onboardingSearchSpinner.style.display = 'block';
    searchTimeout = setTimeout(async () => {
      try {
        const res = await authFetch(`/api/schools/search?q=${encodeURIComponent(q)}`);
        if (res.ok) {
          const data = await res.json();
          renderOnboardingSchoolResults(data.schools || []);
        }
      } catch (e) {
        console.error('School search error:', e);
      } finally {
        el.onboardingSearchSpinner.style.display = 'none';
      }
    }, 280);
  }

  function renderOnboardingSchoolResults(schools) {
    el.onboardingSchoolResults.innerHTML = '';
    if (!schools || schools.length === 0) {
      const emptyDiv = document.createElement('div');
      emptyDiv.style.padding = '12px 14px';
      emptyDiv.style.color = 'var(--text-muted)';
      emptyDiv.style.fontSize = '0.88rem';
      emptyDiv.textContent = 'Keine Schulen für diesen Suchbegriff gefunden.';
      el.onboardingSchoolResults.appendChild(emptyDiv);
      return;
    }

    schools.forEach(school => {
      const card = document.createElement('div');
      card.className = 'onboarding-school-card';
      card.innerHTML = `
        <div class="school-card-title">${escapeHtml(school.displayName)}</div>
        <div class="school-card-address">${escapeHtml(school.address || school.serverUrl || '')}</div>
        <div class="school-card-tags">
          <span class="school-tag">${escapeHtml(school.loginName)}</span>
          <span class="school-tag">${escapeHtml(school.server || 'webuntis.com')}</span>
        </div>
      `;

      card.addEventListener('click', () => {
        selectOnboardingSchool(school);
      });
      el.onboardingSchoolResults.appendChild(card);
    });
  }

  function selectOnboardingSchool(school) {
    state.onboardingSelectedSchool = school;
    el.onboardingSelectedSchoolName.textContent = school.displayName;
    el.onboardingSelectedSchoolDetails.textContent = `${school.serverUrl || school.server} · Kürzel: ${school.loginName}`;
    
    el.onboardingSchoolResults.innerHTML = '';
    el.onboardingCredentialsForm.style.display = 'flex';
    el.onboardingErrorMsg.style.display = 'none';
    el.onboardingUsernameInput.focus();
  }

  async function handleOnboardingSubmit(e) {
    e.preventDefault();
    if (!state.onboardingSelectedSchool) return;

    el.onboardingConnectBtn.disabled = true;
    el.onboardingConnectBtn.querySelector('span').textContent = 'Verbindung wird hergestellt...';
    el.onboardingErrorMsg.style.display = 'none';

    const school = state.onboardingSelectedSchool;
    let serverURL = school.serverUrl || school.server;
    if (!serverURL.startsWith('http://') && !serverURL.startsWith('https://')) {
      serverURL = 'https://' + serverURL;
    }

    const payload = {
      name: school.displayName,
      school: school.loginName,
      server: serverURL,
      username: el.onboardingUsernameInput.value.trim(),
      password: el.onboardingPasswordInput.value,
      setActive: true
    };

    try {
      const res = await authFetch('/api/profiles', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      const data = await res.json();

      if (res.ok && data.success) {
        showToast('Schule erfolgreich verbunden!');
        showOnboardingScreen(false);
        await loadStatus();
        await loadProfiles();
        await loadClasses();
        updateDateDisplay();
        await loadTimetable();
      } else {
        el.onboardingErrorMsg.textContent = data.message || 'Verbindung zu WebUntis fehlgeschlagen.';
        el.onboardingErrorMsg.style.display = 'block';
      }
    } catch (err) {
      el.onboardingErrorMsg.textContent = 'Netzwerkfehler: ' + err.message;
      el.onboardingErrorMsg.style.display = 'block';
    } finally {
      el.onboardingConnectBtn.disabled = false;
      el.onboardingConnectBtn.querySelector('span').textContent = 'Verbindung herstellen & Stundenplan anzeigen';
    }
  }

  // Load Profiles from SQLite
  async function loadProfiles() {
    try {
      const res = await authFetch('/api/profiles');
      if (res.ok) {
        const data = await res.json();
        state.profiles = data.profiles || [];
        state.activeProfileId = data.activeProfileId || '';
        renderProfileQuickList();
        renderSettingsProfileCards();
      }
    } catch (e) {
      console.warn('[Profiles] Fehler beim Laden:', e);
    }
  }

  function renderProfileQuickList() {
    el.profileQuickList.innerHTML = '';
    state.profiles.forEach(p => {
      const item = document.createElement('div');
      const isActive = p.id === state.activeProfileId;
      item.className = `profile-quick-item ${isActive ? 'active' : ''}`;
      item.innerHTML = `
        <div class="avatar-badge" style="width:28px;height:28px;font-size:0.75rem;">
          ${escapeHtml(getAvatarInitials(p.name, p.school))}
        </div>
        <div class="profile-quick-meta">
          <span class="profile-quick-name">${escapeHtml(p.name)}</span>
          <span class="profile-quick-sub">${escapeHtml(p.username || 'anonym')} · ${escapeHtml(p.school)}</span>
        </div>
        ${isActive ? `
          <svg class="profile-check-icon" viewBox="0 0 24 24" fill="currentColor">
            <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
          </svg>` : ''}
      `;

      item.addEventListener('click', () => switchProfile(p.id));
      el.profileQuickList.appendChild(item);
    });
  }

  function renderSettingsProfileCards() {
    el.profileCardsList.innerHTML = '';
    state.profiles.forEach(p => {
      const card = document.createElement('div');
      const isActive = p.id === state.activeProfileId;
      card.className = `profile-card ${isActive ? 'active' : ''}`;
      card.innerHTML = `
        <div style="display:flex;align-items:center;gap:10px;">
          <div class="avatar-badge" style="width:30px;height:30px;font-size:0.8rem;">
            ${escapeHtml(getAvatarInitials(p.name, p.school))}
          </div>
          <div class="profile-card-info">
            <span class="profile-card-name">${escapeHtml(p.name)}</span>
            <span class="profile-card-sub">Benutzer: ${escapeHtml(p.username || 'anonym')} · Schule: ${escapeHtml(p.school)}</span>
          </div>
        </div>
        ${isActive ? '<span class="profile-active-tag">Aktiv</span>' : '<span style="font-size:0.75rem;opacity:0.7;">Wechseln</span>'}
      `;

      card.addEventListener('click', () => switchProfile(p.id));
      el.profileCardsList.appendChild(card);
    });
  }

  async function switchProfile(profileId) {
    if (profileId === state.activeProfileId) return;
    setLoading(true, 'Profil wird gewechselt...');
    try {
      const res = await authFetch('/api/profiles/switch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ profileId })
      });
      const data = await res.json();
      if (data.success) {
        showToast(data.message || 'Profil gewechselt');
        state.activeProfileId = profileId;
        el.profileDropdownWrapper.classList.remove('open');
        closeSettings();
        await loadStatus();
        await loadProfiles();
        await loadClasses();
        await loadTimetable();
      } else {
        showToast('Fehler beim Profilwechsel: ' + data.message);
      }
    } catch (e) {
      showToast('Netzwerkfehler beim Profilwechsel');
    } finally {
      setLoading(false);
    }
  }

  // Load Classes (Cache-First from SQLite)
  async function loadClasses() {
    try {
      const res = await authFetch('/api/classes');
      if (res.ok) {
        const classes = await res.json();
        state.classes = Array.isArray(classes) ? classes : [];
        filterClasses();

        // Auto-select class if none currently selected
        if ((!state.selectedClassId || state.selectedClassId === 0) && state.classes.length > 0) {
          selectClass(state.classes[0].id, state.classes[0].name, false);
        } else {
          updateCurrentClassLabel();
        }
      }
    } catch (e) {
      console.error('[Classes] Fehler:', e);
    }
  }

  function filterClasses() {
    const term = state.classSearchTerm.trim().toLowerCase();
    if (!term) {
      state.filteredClasses = [...state.classes];
    } else {
      state.filteredClasses = state.classes.filter(k => {
        const nameMatch = k.name && k.name.toLowerCase().includes(term);
        const longMatch = k.longName && k.longName.toLowerCase().includes(term);
        return nameMatch || longMatch;
      });
    }
    renderClassDropdown();
  }

  function renderClassDropdown() {
    el.classListContainer.innerHTML = '';
    const total = state.classes.length;
    const filtered = state.filteredClasses.length;

    if (total === filtered) {
      el.classCounterBadge.textContent = `${total} Klassen`;
    } else {
      el.classCounterBadge.textContent = `${filtered} von ${total} Klassen`;
    }

    if (state.filteredClasses.length === 0) {
      const emptyDiv = document.createElement('div');
      emptyDiv.style.padding = '14px 16px';
      emptyDiv.style.fontSize = '0.85rem';
      emptyDiv.style.color = 'var(--text-secondary)';
      emptyDiv.textContent = 'Keine passende Klasse gefunden.';
      el.classListContainer.appendChild(emptyDiv);
      return;
    }

    state.filteredClasses.forEach(k => {
      const btn = document.createElement('button');
      btn.type = 'button';
      const isActive = k.id === state.selectedClassId;
      btn.className = `class-item-btn ${isActive ? 'active' : ''}`;
      btn.innerHTML = `
        <span style="font-weight: 600;">${escapeHtml(k.name)}</span>
        <span style="font-size:0.75rem;opacity:0.7;max-width:160px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">
          ${escapeHtml(k.longName || '')}
        </span>
      `;

      btn.addEventListener('click', () => {
        selectClass(k.id, k.name, true);
      });
      el.classListContainer.appendChild(btn);
    });
  }

  function updateCurrentClassLabel() {
    if (state.selectedClassName) {
      el.currentClassLabel.textContent = `Klasse ${state.selectedClassName}`;
    } else {
      el.currentClassLabel.textContent = 'Klasse wählen';
    }
  }

  async function selectClass(id, name, reload = true) {
    state.selectedClassId = id;
    state.selectedClassName = name;
    updateCurrentClassLabel();
    el.classDropdownWrapper.classList.remove('open');
    renderClassDropdown();

    // Persist selected class to SQLite
    authFetch('/api/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ selected_class_id: id, selected_class_name: name })
    });

    if (reload) {
      await loadTimetable();
    }
  }

  // Timetable Retrieval & Rendering (Zero-Lag Cache-First)
  async function loadTimetable(forceRefresh = false) {
    if (!state.selectedClassId) return;
    setLoading(true, 'Stundenplan wird geladen...');
    try {
      const dateStr = formatDateISO(state.currentDate);
      const url = `/api/timetable?classId=${state.selectedClassId}&date=${dateStr}&view=${state.currentView}${forceRefresh ? '&force=true' : ''}`;
      const res = await authFetch(url);
      if (!res.ok) {
        throw new Error('Fehler beim Laden des Stundenplans');
      }
      state.lessons = await res.json();
      renderCurrentView();
      updateLiveTimeline();
    } catch (e) {
      console.error('[Timetable] Fehler:', e);
      showToast('Konnte Stundenplan nicht abrufen');
      renderEmptyState();
    } finally {
      setLoading(false);
    }
  }

  function setLoading(val, msg = 'Laden...') {
    state.loading = val;
    el.loadingMessage.textContent = msg;
    el.loadingOverlay.classList.toggle('visible', val);
  }

  function renderCurrentView() {
    if (state.currentView === 'day') {
      el.dayViewContainer.classList.add('active');
      el.weekViewContainer.classList.remove('active');
      renderDayView();
    } else {
      el.dayViewContainer.classList.remove('active');
      el.weekViewContainer.classList.add('active');
      renderWeekView();
    }
  }

  function renderEmptyState() {
    el.dayLessonsList.innerHTML = '';
    el.weekGridHeader.innerHTML = '';
    el.weekGridBody.innerHTML = '';
    el.emptyState.style.display = 'flex';
    el.liveTimelineMarker.style.display = 'none';

    const curISO = formatDateISO(state.currentDate);
    if (curISO < '2026-09-07') {
      el.emptyStateTitle.textContent = 'Noch keine Schulstunden';
      el.emptyStateDesc.textContent = 'Der reguläre Unterricht am Berufskolleg beginnt ab Montag, 07. September 2026.';
      el.jumpToNextWeekBtn.style.display = 'inline-block';
      el.jumpToNextWeekBtn.textContent = 'Zu den ersten Schulstunden (Mo, 07.09.2026)';
      el.jumpToNextWeekBtn.onclick = () => {
        state.currentDate = new Date('2026-09-07T10:00:00');
        updateDateDisplay();
        loadTimetable();
      };
    } else {
      el.emptyStateTitle.textContent = 'Kein Unterricht';
      el.emptyStateDesc.textContent = 'Für diesen Zeitraum sind keine Unterrichtsstunden eingetragen.';
      el.jumpToNextWeekBtn.style.display = 'none';
    }
  }

  function renderDayView() {
    el.emptyState.style.display = 'none';
    el.dayLessonsList.innerHTML = '';

    if (!state.lessons || state.lessons.length === 0) {
      renderEmptyState();
      return;
    }

    let lastEndTime = 0;
    state.lessons.forEach(lesson => {
      const startParts = lesson.startTimeStr.split(':');
      const currentStartMinutes = parseInt(startParts[0], 10) * 60 + parseInt(startParts[1], 10);

      if (lastEndTime > 0 && currentStartMinutes - lastEndTime >= 10) {
        const breakMinutes = currentStartMinutes - lastEndTime;
        const breakEl = document.createElement('div');
        breakEl.className = 'break-item';
        const breakTitle = breakMinutes >= 35 ? 'Mittagspause' : 'Pause';
        breakEl.innerHTML = `<span class="break-line"></span><span>${breakTitle} (${breakMinutes} min)</span><span class="break-line"></span>`;
        el.dayLessonsList.appendChild(breakEl);
      }

      const endParts = lesson.endTimeStr.split(':');
      lastEndTime = parseInt(endParts[0], 10) * 60 + parseInt(endParts[1], 10);

      const card = createDayLessonCard(lesson);
      el.dayLessonsList.appendChild(card);
    });
  }

  function createDayLessonCard(l) {
    const card = document.createElement('div');
    card.className = `lesson-card ${l.isCancelled ? 'cancelled' : ''} ${l.isSubstitution ? 'substitution' : ''} ${l.isRoomChange ? 'room-change' : ''}`;
    card.setAttribute('data-start', l.startTimeStr);
    card.setAttribute('data-end', l.endTimeStr);

    let badgesHtml = '';
    if (l.isCancelled) {
      badgesHtml += `<span class="badge-tag badge-cancel">Entfall</span>`;
    } else if (l.isSubstitution) {
      badgesHtml += `<span class="badge-tag badge-subst">Vertretung</span>`;
    }
    if (l.isRoomChange && !l.isCancelled) {
      badgesHtml += `<span class="badge-tag badge-room">Raumänderung</span>`;
    }

    let teacherDisplay = escapeHtml(l.teacher || 'N/A');
    if (l.originalTeacher && l.originalTeacher !== l.teacher) {
      teacherDisplay = `<span class="pill-original">${escapeHtml(l.originalTeacher)}</span><span class="pill-highlight">${escapeHtml(l.teacher)}</span>`;
    }

    let roomDisplay = escapeHtml(l.room || 'N/A');
    if (l.originalRoom && l.originalRoom !== l.room) {
      roomDisplay = `<span class="pill-original">${escapeHtml(l.originalRoom)}</span><span class="pill-highlight">${escapeHtml(l.room)}</span>`;
    }

    let noticeHtml = '';
    if (l.substText) {
      const noticeClass = l.isCancelled ? 'notice-cancel' : (l.isSubstitution ? 'notice-subst' : 'notice-room');
      noticeHtml = `
        <div class="card-notice-banner ${noticeClass}">
          <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
          <span>${escapeHtml(l.substText)}</span>
        </div>`;
    }

    card.innerHTML = `
      <div class="lesson-accent-bar" style="background-color: ${l.isCancelled ? '#ba1a1a' : l.color};"></div>
      <div class="lesson-card-body">
        <div class="card-top-row">
          <span class="period-badge">${escapeHtml(l.period)}</span>
          <span class="time-range-text">${escapeHtml(l.timeRange)}</span>
        </div>
        <div class="card-main-row">
          <div class="subject-group">
            <span class="subject-code" style="color: ${l.isCancelled ? '#ba1a1a' : 'inherit'};">${escapeHtml(l.subject)}</span>
            <span class="subject-long">${escapeHtml(l.subjectLong)}</span>
          </div>
          <div class="badges-group">${badgesHtml}</div>
        </div>
        <div class="card-bottom-row">
          <div class="info-pill" title="Lehrkraft">
            <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/></svg>
            <span>${teacherDisplay}</span>
          </div>
          <div class="info-pill" title="Raum">
            <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/></svg>
            <span>${roomDisplay}</span>
          </div>
          <div class="info-pill" title="Klasse">
            <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 3L1 9l4 2.18v6L12 21l7-3.82v-6l2-1.09V17h2V9L12 3z"/></svg>
            <span>${escapeHtml(l.class || state.selectedClassName)}</span>
          </div>
        </div>
        ${noticeHtml}
      </div>
    `;

    card.addEventListener('click', () => openDetailSheet(l));
    return card;
  }

  // Week View
  function renderWeekView() {
    el.emptyState.style.display = 'none';
    el.liveTimelineMarker.style.display = 'none';
    el.weekGridHeader.innerHTML = '';
    el.weekGridBody.innerHTML = '';

    if (!state.lessons || state.lessons.length === 0) {
      renderEmptyState();
      return;
    }

    const monday = getMondayOfWeek(state.currentDate);
    const dayCols = [];
    const today = new Date();

    for (let i = 0; i < 5; i++) {
      const dayDate = new Date(monday);
      dayDate.setDate(monday.getDate() + i);
      const isToday = isSameDate(dayDate, today);

      const headerCell = document.createElement('div');
      headerCell.className = `week-col-header ${isToday ? 'today' : ''}`;
      headerCell.innerHTML = `
        <span class="day-name">${getGermanWeekday(dayDate)}</span>
        <span class="day-date">${dayDate.getDate()}. ${getGermanMonth(dayDate).substring(0, 3)}</span>
        ${isToday ? '<span class="today-tag">Heute</span>' : ''}
      `;
      el.weekGridHeader.appendChild(headerCell);

      const colBody = document.createElement('div');
      colBody.className = `week-col-body ${isToday ? 'today-col' : ''}`;
      dayCols.push({ dateStr: formatDateISO(dayDate), el: colBody, isToday: isToday });
      el.weekGridBody.appendChild(colBody);
    }

    // Distribute lessons into day columns
    dayCols.forEach(col => {
      const daysLessons = state.lessons.filter(l => l.date === col.dateStr);
      if (daysLessons.length === 0) {
        const noLes = document.createElement('div');
        noLes.className = 'week-no-lessons';
        noLes.textContent = 'Kein Unterricht';
        col.el.appendChild(noLes);
      } else {
        daysLessons.forEach(l => {
          const card = createWeekLessonCard(l);
          col.el.appendChild(card);
        });
      }
    });
  }

  function createWeekLessonCard(l) {
    const card = document.createElement('div');
    card.className = `week-lesson-card ${l.isCancelled ? 'cancelled' : ''} ${l.isSubstitution ? 'substitution' : ''}`;
    card.style.borderLeftColor = l.isCancelled ? '#ba1a1a' : l.color;

    card.innerHTML = `
      <div class="week-card-top">
        <span class="week-subject">${escapeHtml(l.subject)}</span>
        <span class="week-time">${escapeHtml(l.startTimeStr)}</span>
      </div>
      <div class="week-card-bottom">
        <span class="week-teacher">${escapeHtml(l.teacher || '')}</span>
        <span class="week-room">${escapeHtml(l.room || '')}</span>
      </div>
    `;

    card.addEventListener('click', () => openDetailSheet(l));
    return card;
  }

  // Live Time Line (Rote Markierungslinie für aktuelle Uhrzeit)
  function updateLiveTimeline() {
    if (state.currentView !== 'day') {
      el.liveTimelineMarker.style.display = 'none';
      return;
    }

    const now = new Date();
    const isToday = isSameDate(state.currentDate, now);

    if (!isToday || state.lessons.length === 0) {
      el.liveTimelineMarker.style.display = 'none';
      return;
    }

    const nowHours = String(now.getHours()).padStart(2, '0');
    const nowMinutes = String(now.getMinutes()).padStart(2, '0');
    const currentTotalMinutes = now.getHours() * 60 + now.getMinutes();

    el.liveTimelineTimeLabel.textContent = `JETZT ${nowHours}:${nowMinutes}`;

    // Find insertion position among day lesson cards
    const cards = Array.from(el.dayLessonsList.querySelectorAll('.lesson-card'));
    if (cards.length === 0) {
      el.liveTimelineMarker.style.display = 'none';
      return;
    }

    let inserted = false;
    for (let i = 0; i < cards.length; i++) {
      const startStr = cards[i].getAttribute('data-start');
      if (!startStr) continue;
      const parts = startStr.split(':');
      const startMins = parseInt(parts[0], 10) * 60 + parseInt(parts[1], 10);

      if (currentTotalMinutes < startMins) {
        cards[i].parentNode.insertBefore(el.liveTimelineMarker, cards[i]);
        el.liveTimelineMarker.style.display = 'flex';
        inserted = true;
        break;
      }
    }

    if (!inserted) {
      // Append after last card
      el.dayLessonsList.appendChild(el.liveTimelineMarker);
      el.liveTimelineMarker.style.display = 'flex';
    }
  }

  // Lesson Detail Sheet
  function openDetailSheet(l) {
    el.sheetSubjectBadge.textContent = l.subject ? l.subject.substring(0, 3) : 'U';
    el.sheetSubjectBadge.style.backgroundColor = l.isCancelled ? '#feeceb' : l.color;
    el.sheetSubjectBadge.style.color = l.isCancelled ? '#ba1a1a' : l.textColor;

    el.sheetSubjectTitle.textContent = l.subjectLong || l.subject;
    el.sheetTimeTitle.textContent = `${l.timeRange} · ${l.period}`;

    // Status Badges
    el.sheetStatusRow.innerHTML = '';
    if (l.isCancelled) {
      el.sheetStatusRow.innerHTML += `<span class="badge-tag badge-cancel">Entfall</span>`;
    } else if (l.isSubstitution) {
      el.sheetStatusRow.innerHTML += `<span class="badge-tag badge-subst">Vertretung</span>`;
    } else {
      el.sheetStatusRow.innerHTML += `<span class="badge-tag" style="background:var(--surface-container);color:var(--text-secondary);">Regulär</span>`;
    }
    if (l.isRoomChange && !l.isCancelled) {
      el.sheetStatusRow.innerHTML += `<span class="badge-tag badge-room">Raumänderung</span>`;
    }

    // Teacher
    let tText = l.teacherLong ? `${l.teacherLong} (${l.teacher})` : (l.teacher || '-');
    if (l.originalTeacher && l.originalTeacher !== l.teacher) {
      tText = `Vertretung für ${l.originalTeacher}: ${tText}`;
    }
    el.sheetTeacherVal.textContent = tText;

    // Room
    let rText = l.roomLong ? `${l.roomLong} (${l.room})` : (l.room || '-');
    if (l.originalRoom && l.originalRoom !== l.room) {
      rText = `Statt Raum ${l.originalRoom}: ${rText}`;
    }
    el.sheetRoomVal.textContent = rText;

    // Class
    el.sheetClassVal.textContent = l.class || state.selectedClassName || '-';

    // Date
    el.sheetDateTimeVal.textContent = `${l.dayOfWeek}, ${l.date} (${l.timeRange})`;

    // Teaching Content
    if (l.teachingContent) {
      el.sheetTeachingContentBox.textContent = l.teachingContent;
      el.sheetTeachingContentSection.style.display = 'block';
    } else {
      el.sheetTeachingContentSection.style.display = 'none';
    }

    // Homework
    if (l.homeworks && l.homeworks.length > 0) {
      el.sheetHomeworkBox.innerHTML = l.homeworks.map(h => `<p>• ${escapeHtml(h)}</p>`).join('');
      el.sheetHomeworkSection.style.display = 'block';
    } else {
      el.sheetHomeworkSection.style.display = 'none';
    }

    // Notes / Substitution Text
    if (l.notes || l.substText) {
      const combined = [l.substText, l.notes].filter(Boolean).join('\n\n');
      el.sheetNotesBox.textContent = combined;
      el.sheetNotesSection.style.display = 'block';
    } else {
      el.sheetNotesSection.style.display = 'none';
    }

    el.sheetBackdrop.classList.add('open');
    el.lessonDetailSheet.classList.add('open');
  }

  function closeDetailSheet() {
    el.sheetBackdrop.classList.remove('open');
    el.lessonDetailSheet.classList.remove('open');
  }

  // Date Navigation
  function stepDate(direction) {
    const d = new Date(state.currentDate);
    if (state.currentView === 'day') {
      d.setDate(d.getDate() + direction);
      if (d.getDay() === 6) {
        d.setDate(d.getDate() + (direction > 0 ? 2 : -1));
      } else if (d.getDay() === 0) {
        d.setDate(d.getDate() + (direction > 0 ? 1 : -2));
      }
    } else {
      d.setDate(d.getDate() + (direction * 7));
    }
    state.currentDate = d;
    updateDateDisplay();
    loadTimetable();
  }

  function updateDateDisplay() {
    const d = state.currentDate;
    const now = new Date();

    if (state.currentView === 'day') {
      const weekday = getGermanWeekday(d);
      const day = d.getDate();
      const month = getGermanMonth(d);
      const year = d.getFullYear();

      if (isSameDate(d, now)) {
        el.dateDisplayLabel.textContent = `Heute · ${day}. ${month}`;
      } else {
        el.dateDisplayLabel.textContent = `${weekday}, ${day}. ${month} ${year}`;
      }
    } else {
      const mon = getMondayOfWeek(d);
      const fri = getFridayOfWeek(d);
      el.dateDisplayLabel.textContent = `${mon.getDate()}. ${getGermanMonth(mon).substring(0, 3)} - ${fri.getDate()}. ${getGermanMonth(fri).substring(0, 3)} ${fri.getFullYear()}`;
    }

    el.nativeDatePicker.value = formatDateISO(d);
  }

  function updateViewToggle() {
    if (state.currentView === 'day') {
      el.viewDayBtn.classList.add('active');
      el.viewWeekBtn.classList.remove('active');
    } else {
      el.viewDayBtn.classList.remove('active');
      el.viewWeekBtn.classList.add('active');
    }
  }

  // Settings Modal Handlers
  function openSettings() {
    el.settingsStatusMsg.style.display = 'none';
    el.schoolResultsList.innerHTML = '';
    renderSettingsProfileCards();
    el.settingsModalBackdrop.classList.add('open');
    el.profileDropdownWrapper.classList.remove('open');
  }

  function closeSettings() {
    el.settingsModalBackdrop.classList.remove('open');
  }

  async function handleSaveSettings(e) {
    e.preventDefault();
    el.saveSettingsBtn.disabled = true;
    el.saveSettingsBtn.textContent = 'Prüfe Verbindung...';
    el.settingsStatusMsg.style.display = 'none';

    const payload = {
      name: el.cfgProfileName.value.trim() || el.cfgSchool.value.trim(),
      school: el.cfgSchool.value.trim(),
      server: el.cfgServer.value.trim(),
      username: el.cfgUsername.value.trim(),
      password: el.cfgPassword.value,
      setActive: true
    };

    try {
      const res = await authFetch('/api/profiles', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      const data = await res.json();

      if (res.ok && data.success) {
        el.settingsStatusMsg.textContent = data.message || 'Einstellungen gespeichert!';
        el.settingsStatusMsg.className = 'status-message success';
        el.settingsStatusMsg.style.display = 'block';

        // Save theme and default view
        authFetch('/api/settings', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            theme: el.cfgTheme.value,
            default_view: el.cfgDefaultView.value
          })
        });

        applyTheme(el.cfgTheme.value);

        setTimeout(async () => {
          closeSettings();
          await loadStatus();
          await loadProfiles();
          await loadClasses();
          await loadTimetable();
          showToast('Erfolgreich gespeichert!');
        }, 600);
      } else {
        el.settingsStatusMsg.textContent = data.message || 'Fehler beim Verbinden mit WebUntis';
        el.settingsStatusMsg.className = 'status-message error';
        el.settingsStatusMsg.style.display = 'block';
      }
    } catch (err) {
      el.settingsStatusMsg.textContent = 'Netzwerkfehler: ' + err.message;
      el.settingsStatusMsg.className = 'status-message error';
      el.settingsStatusMsg.style.display = 'block';
    } finally {
      el.saveSettingsBtn.disabled = false;
      el.saveSettingsBtn.textContent = 'Verbindung testen & Speichern';
    }
  }

  async function handleClearCache() {
    if (!confirm('Möchten Sie den lokalen Stundenplan-Cache in SQLite leeren?')) return;
    try {
      const res = await authFetch('/api/refresh', { method: 'POST' });
      if (res.ok) {
        showToast('Cache geleert');
        await loadClasses();
        await loadTimetable(true);
      }
    } catch (e) {
      showToast('Fehler beim Leeren des Caches');
    }
  }

  // Setup All Event Listeners
  function setupEventListeners() {
    // Class Dropdown Toggle
    el.classSelectorBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      el.profileDropdownWrapper.classList.remove('open');
      const isOpen = el.classDropdownWrapper.classList.toggle('open');
      if (isOpen) {
        setTimeout(() => el.classSearchInput.focus(), 100);
      }
    });

    // Profile Dropdown Toggle
    el.profileBadgeBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      el.classDropdownWrapper.classList.remove('open');
      el.profileDropdownWrapper.classList.toggle('open');
    });

    document.addEventListener('click', (e) => {
      if (!el.classDropdownWrapper.contains(e.target)) {
        el.classDropdownWrapper.classList.remove('open');
      }
      if (!el.profileDropdownWrapper.contains(e.target)) {
        el.profileDropdownWrapper.classList.remove('open');
      }
    });

    // Add New School from Quick Switcher
    el.addNewProfileQuickBtn.addEventListener('click', () => {
      el.profileDropdownWrapper.classList.remove('open');
      showOnboardingScreen(true);
    });

    // Add New School from Settings Modal
    el.addNewProfileModalBtn.addEventListener('click', () => {
      closeSettings();
      showOnboardingScreen(true);
    });

    // Class Search Input
    el.classSearchInput.addEventListener('input', (e) => {
      state.classSearchTerm = e.target.value;
      el.clearClassSearchBtn.style.display = state.classSearchTerm ? 'block' : 'none';
      filterClasses();
    });

    el.clearClassSearchBtn.addEventListener('click', () => {
      el.classSearchInput.value = '';
      state.classSearchTerm = '';
      el.clearClassSearchBtn.style.display = 'none';
      filterClasses();
      el.classSearchInput.focus();
    });

    // Onboarding Live Search
    el.onboardingSearchInput.addEventListener('input', (e) => {
      handleOnboardingSearch(e.target.value);
    });

    el.onboardingChangeSchoolBtn.addEventListener('click', () => {
      el.onboardingCredentialsForm.style.display = 'none';
      el.onboardingSearchInput.focus();
    });

    el.onboardingLoginFormBox.addEventListener('submit', handleOnboardingSubmit);

    el.toggleOnboardingPwdBtn.addEventListener('click', () => {
      const isPwd = el.onboardingPasswordInput.type === 'password';
      el.onboardingPasswordInput.type = isPwd ? 'text' : 'password';
    });

    // Date Navigation
    el.prevDateBtn.addEventListener('click', () => stepDate(-1));
    el.nextDateBtn.addEventListener('click', () => stepDate(1));

    el.todayBtn.addEventListener('click', () => {
      state.currentDate = new Date();
      updateDateDisplay();
      loadTimetable();
    });

    el.dateDisplayBtn.addEventListener('click', () => {
      el.nativeDatePicker.showPicker ? el.nativeDatePicker.showPicker() : el.nativeDatePicker.focus();
    });

    el.nativeDatePicker.addEventListener('change', (e) => {
      if (e.target.value) {
        state.currentDate = new Date(e.target.value + 'T12:00:00');
        updateDateDisplay();
        loadTimetable();
      }
    });

    // View Toggle
    el.viewDayBtn.addEventListener('click', () => {
      state.currentView = 'day';
      updateViewToggle();
      updateDateDisplay();
      renderCurrentView();
      updateLiveTimeline();
    });

    el.viewWeekBtn.addEventListener('click', () => {
      state.currentView = 'week';
      updateViewToggle();
      updateDateDisplay();
      loadTimetable();
    });

    // Refresh Button (spins smoothly)
    el.refreshBtn.addEventListener('click', async () => {
      el.refreshBtn.style.transform = 'rotate(360deg)';
      el.refreshBtn.style.transition = 'transform 0.5s ease';
      await loadTimetable(true);
      setTimeout(() => {
        el.refreshBtn.style.transform = '';
        el.refreshBtn.style.transition = '';
      }, 500);
      showToast('Stundenplan aktualisiert');
    });

    // Theme Toggle
    el.themeToggleBtn.addEventListener('click', () => {
      const nextTheme = state.theme === 'dark' ? 'light' : 'dark';
      applyTheme(nextTheme);
      authFetch('/api/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ theme: nextTheme })
      });
      showToast(nextTheme === 'dark' ? 'Dark Mode aktiviert' : 'Light Mode aktiviert');
    });

    // Settings Modal
    el.settingsBtn.addEventListener('click', openSettings);
    el.closeSettingsBtn.addEventListener('click', closeSettings);
    el.settingsModalBackdrop.addEventListener('click', (e) => {
      if (e.target === el.settingsModalBackdrop) closeSettings();
    });
    el.settingsForm.addEventListener('submit', handleSaveSettings);
    el.clearCacheBtn.addEventListener('click', handleClearCache);

    // School Search in Settings
    el.searchSchoolBtn.addEventListener('click', async () => {
      const q = el.schoolSearchInput.value.trim();
      if (!q) return;
      el.searchSchoolBtn.disabled = true;
      try {
        const res = await authFetch(`/api/schools/search?q=${encodeURIComponent(q)}`);
        const data = await res.json();
        el.schoolResultsList.innerHTML = '';
        (data.schools || []).forEach(s => {
          const item = document.createElement('div');
          item.className = 'school-result-item';
          item.innerHTML = `
            <div class="school-result-name">${escapeHtml(s.displayName)}</div>
            <div class="school-result-meta">${escapeHtml(s.address || s.serverUrl || '')} · Kürzel: ${escapeHtml(s.loginName)}</div>
          `;
          item.addEventListener('click', () => {
            el.cfgSchool.value = s.loginName;
            el.cfgServer.value = s.serverUrl || s.server;
            el.cfgProfileName.value = s.displayName;
            el.schoolResultsList.innerHTML = '';
            showToast(`Schule '${s.displayName}' übernommen`);
          });
          el.schoolResultsList.appendChild(item);
        });
      } finally {
        el.searchSchoolBtn.disabled = false;
      }
    });

    // Toggle Password in Settings
    el.togglePasswordBtn.addEventListener('click', () => {
      const isPwd = el.cfgPassword.type === 'password';
      el.cfgPassword.type = isPwd ? 'text' : 'password';
    });

    // Detail Sheet Close
    el.sheetCloseBtn.addEventListener('click', closeDetailSheet);
    el.sheetBackdrop.addEventListener('click', closeDetailSheet);

    // Keyboard Shortcuts (Arrow keys, T, D, W, Esc)
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        closeDetailSheet();
        closeSettings();
        el.classDropdownWrapper.classList.remove('open');
        el.profileDropdownWrapper.classList.remove('open');
        if (state.onboardingSelectedSchool && !state.needsOnboarding) {
          showOnboardingScreen(false);
        }
      } else if (e.target.tagName !== 'INPUT' && e.target.tagName !== 'SELECT' && e.target.tagName !== 'TEXTAREA') {
        if (e.key === 'ArrowLeft') {
          stepDate(-1);
        } else if (e.key === 'ArrowRight') {
          stepDate(1);
        } else if (e.key === 't' || e.key === 'T') {
          state.currentDate = new Date();
          updateDateDisplay();
          loadTimetable();
        } else if (e.key === 'd' || e.key === 'D') {
          state.currentView = 'day';
          updateViewToggle();
          updateDateDisplay();
          renderCurrentView();
          updateLiveTimeline();
        } else if (e.key === 'w' || e.key === 'W') {
          state.currentView = 'week';
          updateViewToggle();
          updateDateDisplay();
          loadTimetable();
        }
      }
    });
  }

  function escapeHtml(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  // Launch on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
