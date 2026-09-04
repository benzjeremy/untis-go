/* ==========================================================================
   UNTIS DESKTOP APP - CLIENT APPLICATION LOGIC
   ========================================================================== */

(function () {
  'use strict';

  // State
  const state = {
    token: '',
    currentView: 'dashboard',
    status: null,
    currentDate: new Date(),
    timetableMode: 'day', // 'day' or 'week'
    
    // Other timetables tab & selected item
    otherTab: 'CLASS', // 'CLASS', 'TEACHER', 'ROOM'
    otherSelectedId: null,
    otherSelectedName: '',
    
    classes: [],
    teachers: [],
    rooms: [],
    homework: [],
    absences: [],
    messages: [],
    profiles: [],
    
    hwFilter: 'all', // 'all', 'open', 'completed'
  };

  // Extract Session Token from URL or storage
  function initToken() {
    const urlParams = new URLSearchParams(window.location.search);
    let token = urlParams.get('token');
    if (token) {
      sessionStorage.setItem('untis_session_token', token);
    } else {
      token = sessionStorage.getItem('untis_session_token') || '';
    }
    state.token = token;
  }

  // API Request Helper
  async function apiFetch(endpoint, options = {}) {
    const headers = options.headers || {};
    if (state.token) {
      headers['X-Session-Token'] = state.token;
    }
    
    let url = endpoint;
    if (state.token && !url.includes('token=')) {
      const sep = url.includes('?') ? '&' : '?';
      url = `${url}${sep}token=${encodeURIComponent(state.token)}`;
    }

    try {
      const resp = await fetch(url, {
        ...options,
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          ...headers,
        },
      });

      if (resp.status === 401) {
        showToast('Sitzung abgelaufen oder ungültiges Token', 'error');
        return null;
      }

      return await resp.json();
    } catch (err) {
      console.error(`API Error on ${endpoint}:`, err);
      return null;
    }
  }

  // Formatting Helpers
  function formatDateISO(d) {
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  function formatGermanDate(d) {
    const days = ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag'];
    const months = ['Jan.', 'Feb.', 'März', 'Apr.', 'Mai', 'Juni', 'Juli', 'Aug.', 'Sept.', 'Okt.', 'Nov.', 'Dez.'];
    return `${days[d.getDay()]}, ${d.getDate()}. ${months[d.getMonth()]} ${d.getFullYear()}`;
  }

  function showLoading(show, text = 'Wird geladen...') {
    const overlay = document.getElementById('desktopLoadingOverlay');
    const textEl = document.getElementById('desktopLoadingText');
    if (overlay) {
      overlay.style.display = show ? 'flex' : 'none';
      if (textEl) textEl.textContent = text;
    }
  }

  function showToast(message, type = 'success') {
    const container = document.getElementById('desktopToastContainer');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `desktop-toast ${type}`;
    toast.textContent = message;
    container.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateY(8px)';
      setTimeout(() => toast.remove(), 300);
    }, 3200);
  }

  // ==================== VIEW SWITCHING ====================
  function switchView(viewName) {
    if (viewName === 'profiles') {
      openProfilesModal();
      return;
    }
    state.currentView = viewName;

    // Update Nav buttons
    document.querySelectorAll('.sidebar-nav .nav-item').forEach((item) => {
      if (item.getAttribute('data-view') === viewName) {
        item.classList.add('active');
      } else {
        item.classList.remove('active');
      }
    });

    // Update Panes
    document.querySelectorAll('.desktop-view-pane').forEach((pane) => {
      pane.classList.remove('active');
    });

    const targetMap = {
      'dashboard': 'viewPaneDashboard',
      'own-timetable': 'viewPaneOwnTimetable',
      'other-timetables': 'viewPaneOtherTimetables',
      'homework': 'viewPaneHomework',
      'absences': 'viewPaneAbsences',
      'messages': 'viewPaneMessages',
      'profiles': 'viewPaneProfiles',
    };

    const targetPaneId = targetMap[viewName];
    if (targetPaneId) {
      const pane = document.getElementById(targetPaneId);
      if (pane) pane.classList.add('active');
    }

    // Topbar header updates
    const titleEl = document.getElementById('topbarViewTitle');
    const descEl = document.getElementById('topbarViewDesc');
    const ttControls = document.getElementById('topbarTimetableControls');

    const metaMap = {
      'dashboard': { title: 'Übersicht', desc: 'Willkommen in deiner persönlichen Untis-Zentrale' },
      'own-timetable': { title: 'Mein Stundenplan', desc: 'Persönlicher Schüler-Stundenplan' },
      'other-timetables': { title: 'Weitere Stundenpläne', desc: 'Klassen, Lehrkräfte und Fachräume' },
      'homework': { title: 'Hausaufgaben', desc: 'WebUntis Aufgaben & eigene Notizen' },
      'absences': { title: 'Abwesenheiten', desc: 'Fehlzeiten & Krankmeldungen' },
      'messages': { title: 'Mitteilungen', desc: 'Schulposteingang & Durchsagen' },
      'profiles': { title: 'Profile & Schulen', desc: 'Schulzugänge verwalten und hinzufügen' },
    };

    if (metaMap[viewName]) {
      if (titleEl) titleEl.textContent = metaMap[viewName].title;
      if (descEl) descEl.textContent = metaMap[viewName].desc;
    }

    // Toggle topbar date controls
    if (ttControls) {
      ttControls.style.display = (viewName === 'own-timetable' || viewName === 'other-timetables') ? 'flex' : 'none';
    }

    // Load data for view
    switch (viewName) {
      case 'dashboard':
        loadDashboard();
        break;
      case 'own-timetable':
        loadOwnTimetable();
        break;
      case 'other-timetables':
        loadOtherTimetablesTab();
        break;
      case 'homework':
        loadHomework();
        break;
      case 'absences':
        loadAbsences();
        break;
      case 'messages':
        loadMessages();
        break;
      case 'profiles':
        loadProfiles();
        break;
    }
  }

  // ==================== INITIALIZATION ====================
  async function initApp() {
    initToken();
    setupEventListeners();
    setupTheme();

    // Check system status
    const status = await apiFetch('/api/status');
    if (!status) return;
    state.status = status;

    // Update user info in sidebar
    const schoolNameEl = document.getElementById('sidebarSchoolName');
    const userNameEl = document.getElementById('sidebarUserName');
    const userAvatarEl = document.getElementById('sidebarUserAvatar');
    const heroSchoolEl = document.getElementById('dashHeroSchool');

    const school = status.school || 'WebUntis';
    const displayName = status.displayName || status.profileName || status.username || 'Benutzer';

    if (schoolNameEl) schoolNameEl.textContent = school;
    if (heroSchoolEl) heroSchoolEl.textContent = school;
    if (userNameEl) userNameEl.textContent = displayName;
    if (userAvatarEl) {
      const parts = displayName.split(' ');
      const initials = parts.map(p => p[0]).join('').slice(0, 2).toUpperCase();
      userAvatarEl.textContent = initials || 'U';
    }

    // Dynamic window title: Untis Stundenplan - <Schule> - <Schülername>
    if (school && displayName) {
      document.title = `Untis Stundenplan - ${school} - ${displayName}`;
    } else if (school) {
      document.title = `Untis Stundenplan - ${school}`;
    } else {
      document.title = 'Untis Stundenplan';
    }

    // Automatic update check in background
    checkForSoftwareUpdates(true);

    if (status.needsOnboarding) {
      openProfilesModal();
      return;
    }

    // Load initial dashboard
    updateDateDisplay();
    switchView('dashboard');
  }

  // ==================== DASHBOARD MODULE ====================
  async function loadDashboard() {
    const data = await apiFetch('/api/dashboard');
    if (!data) return;

    // Greeting & Hero
    const greetingEl = document.getElementById('dashHeroGreeting');
    const dateEl = document.getElementById('dashHeroDate');
    if (greetingEl) greetingEl.textContent = data.greeting || 'Guten Tag';
    if (dateEl) dateEl.textContent = data.dateFormatted || formatGermanDate(new Date());

    // Badges on Sidebar
    const hwBadge = document.getElementById('sidebarHwBadge');
    const msgBadge = document.getElementById('sidebarMsgBadge');
    const absBadge = document.getElementById('sidebarAbsBadge');

    if (hwBadge) {
      if (data.openHomeworkCount > 0) {
        hwBadge.textContent = data.openHomeworkCount;
        hwBadge.style.display = 'inline-block';
      } else {
        hwBadge.style.display = 'none';
      }
    }

    // Mitteilungen: calculate unread count based on read message IDs!
    const readIds = getReadMessageIds();
    let unreadCount = 0;
    if (data.recentMessages && data.recentMessages.length > 0) {
      unreadCount = data.recentMessages.filter(m => !readIds.includes(String(m.id))).length;
    } else if (data.messagesCount > 0) {
      unreadCount = Math.max(0, data.messagesCount - readIds.length);
    }

    if (msgBadge) {
      if (unreadCount > 0) {
        msgBadge.textContent = unreadCount;
        msgBadge.style.display = 'inline-block';
      } else {
        msgBadge.textContent = '0';
        msgBadge.style.display = 'none';
      }
    }

    if (absBadge && data.absencesSummary) {
      if (data.absencesSummary.total > 0) {
        absBadge.textContent = data.absencesSummary.total;
        absBadge.style.display = 'inline-block';
      } else {
        absBadge.style.display = 'none';
      }
    }

    // Metric: Next Lesson
    const nextSubj = document.getElementById('dashNextLessonSubject');
    const nextRoom = document.getElementById('dashNextLessonRoom');
    const nextTime = document.getElementById('dashNextLessonTime');
    const nextSubtext = document.getElementById('dashNextLessonSubtext');

    if (data.nextLesson) {
      const l = data.nextLesson;
      if (nextSubj) nextSubj.textContent = l.subjectLong || l.subject;
      if (nextRoom) nextRoom.textContent = `Raum: ${l.room || 'k.A.'}`;
      if (nextTime) nextTime.textContent = l.startTimeStr;
      if (nextSubtext) {
        if (data.isUpcomingSchoolDay && data.upcomingDayLabel) {
          nextSubtext.textContent = `${data.upcomingDayLabel} · ${l.period} · Lehrkraft: ${l.teacher || 'k.A.'}`;
        } else {
          nextSubtext.textContent = `${l.period} · Lehrkraft: ${l.teacher || 'k.A.'}`;
        }
      }
    } else {
      if (nextSubj) nextSubj.textContent = 'Kein Unterricht';
      if (nextRoom) nextRoom.textContent = '';
      if (nextTime) nextTime.textContent = '--:--';
      if (nextSubtext) nextSubtext.textContent = 'Der heutige Schultag ist beendet oder unterrichtsfrei.';
    }

    // Metric: Homework
    const hwCountEl = document.getElementById('dashHwCount');
    const hwHeadlineEl = document.getElementById('dashHwHeadline');
    if (hwCountEl) hwCountEl.textContent = `${data.openHomeworkCount} offen`;
    if (hwHeadlineEl) {
      hwHeadlineEl.textContent = data.openHomeworkCount > 0 ? `${data.openHomeworkCount} Aufgaben zu erledigen` : 'Alles erledigt!';
    }

    // Metric: Messages
    const msgCountEl = document.getElementById('dashMsgCount');
    const msgHeadlineEl = document.getElementById('dashMsgHeadline');
    if (msgCountEl) msgCountEl.textContent = `${unreadCount} ungelesen`;
    if (msgHeadlineEl) {
      msgHeadlineEl.textContent = unreadCount > 0 ? `${unreadCount} neue Mitteilungen` : 'Keine ungelesenen Nachrichten';
    }

    // Metric: Absences
    const absCountEl = document.getElementById('dashAbsCount');
    const absExcEl = document.getElementById('dashAbsExcused');
    const absUnexcEl = document.getElementById('dashAbsUnexcused');
    if (data.absencesSummary) {
      if (absCountEl) absCountEl.textContent = `${data.absencesSummary.total} Einträge`;
      if (absExcEl) absExcEl.textContent = `${data.absencesSummary.excused} Entschuldigt`;
      if (absUnexcEl) absUnexcEl.textContent = `${data.absencesSummary.unexcused} Unentschuldigt`;
    }

    // Today's Lessons List (or upcoming day)
    renderDashboardLessons(data.todayLessons || [], data.isUpcomingSchoolDay, data.upcomingDayLabel);

    // Homework Preview List
    renderDashboardHomework(data.openHomework || []);

    // Messages Preview List
    renderDashboardMessages(data.recentMessages || []);
  }

  function renderDashboardLessons(lessons, isUpcomingDay, upcomingLabel) {
    const list = document.getElementById('dashTodayLessonsList');
    if (!list) return;

    if (!lessons || lessons.length === 0) {
      list.innerHTML = '<div class="empty-inline-state">Heute steht kein planmäßiger Unterricht an.</div>';
      return;
    }

    const headerNote = isUpcomingDay && upcomingLabel
      ? `<div style="font-size:12px; font-weight:700; color:var(--accent-primary); margin-bottom:10px; display:flex; align-items:center; gap:6px;">
           <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>
           ${escapeHTML(upcomingLabel)}
         </div>`
      : '';

    list.innerHTML = headerNote + lessons.map(l => {
      const color = l.color || '#ff7a00';
      const statusBadge = l.isCancelled ? '<span class="status-pill cancelled">Ausfall</span>' : (l.isSubstitution ? '<span class="status-pill substitution">Vertretung</span>' : '');
      return `
        <div class="dash-lesson-item" onclick="openLessonDetailModal(${escapeHTML(JSON.stringify(l))})">
          <div class="lesson-time-col">
            <span class="l-time-range">${l.timeRange}</span>
            <span class="l-period">${l.period}</span>
          </div>
          <div class="lesson-color-bar" style="background-color:${color};"></div>
          <div class="lesson-main-col">
            <div class="l-subject-row">
              <span class="l-subject">${l.subject}</span>
              ${l.room ? `<span class="l-room-tag">${l.room}</span>` : ''}
              ${statusBadge}
            </div>
            <span class="l-teacher">${l.teacherLong || l.teacher || ''}</span>
          </div>
        </div>
      `;
    }).join('');
  }

  function renderDashboardHomework(homeworks) {
    const list = document.getElementById('dashHwList');
    if (!list) return;

    if (!homeworks || homeworks.length === 0) {
      list.innerHTML = '<div class="empty-inline-state">Keine offenen Hausaufgaben eingetragen.</div>';
      return;
    }

    list.innerHTML = homeworks.map(h => `
      <div class="homework-card" style="padding:10px 14px; margin-bottom:6px;">
        <div class="custom-checkbox ${h.completed ? 'checked' : ''}" onclick="toggleHomeworkComplete('${h.id}', ${!h.completed})">
          ${h.completed ? '✓' : ''}
        </div>
        <div class="hw-info">
          <div class="hw-header-line">
            <span class="hw-subject-badge">${h.subject}</span>
            <span class="hw-due-pill">${h.dueDate ? `Fällig: ${h.dueDate}` : ''}</span>
          </div>
          <div class="hw-desc">${escapeHTML(h.description)}</div>
        </div>
      </div>
    `).join('');
  }

  function renderDashboardMessages(messages) {
    const list = document.getElementById('dashMsgList');
    if (!list) return;

    if (!messages || messages.length === 0) {
      list.innerHTML = '<div class="empty-inline-state">Keine neuen Mitteilungen.</div>';
      return;
    }

    list.innerHTML = messages.map(m => `
      <div class="message-card" style="padding:12px 14px; margin-bottom:6px;" onclick="openMessageDetailModal(${m.id})">
        <div class="msg-header-row">
          <span class="msg-sender-chip">${m.senderName || 'Schule'}</span>
          <span class="msg-date-str">${formatDateTime(m.sentDateTime)}</span>
        </div>
        <div class="msg-subject-line" style="font-size:13px;">${escapeHTML(m.subject)}</div>
      </div>
    `).join('');
  }

  // ==================== MEIN STUNDENPLAN MODULE ====================
  async function loadOwnTimetable() {
    showLoading(true, 'Lade deinen persönlichen Stundenplan...');
    const dateStr = formatDateISO(state.currentDate);
    const view = state.timetableMode;

    const lessons = await apiFetch(`/api/timetable/own?date=${dateStr}&view=${view}`);
    showLoading(false);

    const dayContainer = document.getElementById('ownDayViewContainer');
    const weekContainer = document.getElementById('ownWeekViewContainer');
    const emptyState = document.getElementById('ownEmptyState');

    if (!lessons || lessons.length === 0) {
      if (dayContainer) dayContainer.style.display = 'none';
      if (weekContainer) weekContainer.style.display = 'none';
      if (emptyState) emptyState.style.display = 'block';
      return;
    }

    if (emptyState) emptyState.style.display = 'none';

    if (view === 'day') {
      if (dayContainer) dayContainer.style.display = 'block';
      if (weekContainer) weekContainer.style.display = 'none';
      renderDayTimetable(lessons, 'ownDayLessonsList');
    } else {
      if (dayContainer) dayContainer.style.display = 'none';
      if (weekContainer) weekContainer.style.display = 'block';
      renderWeekTimetable(lessons, 'ownWeekGridHeader', 'ownWeekGridBody');
    }

    updateLiveTimeMarker();
  }

  function renderDayTimetable(lessons, targetElementId) {
    const container = document.getElementById(targetElementId);
    if (!container) return;

    container.innerHTML = lessons.map(l => {
      const color = l.color || '#ff7a00';
      const statusBadge = l.isCancelled ? '<span class="status-pill cancelled">Ausfall</span>' : (l.isSubstitution ? '<span class="status-pill substitution">Vertretung</span>' : '');

      return `
        <div class="lesson-card" onclick="openLessonDetailModal(${escapeHTML(JSON.stringify(l))})">
          <div class="lesson-left-group">
            <div class="lesson-card-badge" style="background-color:${color}; color:${l.textColor || '#ffffff'};">
              ${l.subject.slice(0, 4)}
            </div>
            <div class="lesson-info-group">
              <span class="lesson-card-title">${l.subjectLong || l.subject}</span>
              <span class="lesson-card-meta">
                ${l.teacher ? `Lehrkraft: <strong>${l.teacherLong || l.teacher}</strong>` : ''} 
                ${l.room ? `· Raum: <strong>${l.room}</strong>` : ''}
              </span>
              ${statusBadge}
            </div>
          </div>
          <div class="lesson-right-group">
            <span class="lesson-card-time">${l.timeRange}</span>
            <span class="lesson-card-period">${l.period}</span>
          </div>
        </div>
      `;
    }).join('');
  }

  function renderWeekTimetable(lessons, headerId, bodyId) {
    const headerEl = document.getElementById(headerId);
    const bodyEl = document.getElementById(bodyId);
    if (!headerEl || !bodyEl) return;

    // Compute Monday of active week
    const target = new Date(state.currentDate);
    const dayOfWeek = target.getDay();
    const diff = target.getDate() - dayOfWeek + (dayOfWeek === 0 ? -6 : 1);
    const monday = new Date(target.setDate(diff));

    const weekDays = [];
    for (let i = 0; i < 5; i++) {
      const d = new Date(monday);
      d.setDate(monday.getDate() + i);
      weekDays.push(d);
    }

    const dayNames = ['Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag'];
    const todayStr = formatDateISO(new Date());

    // Header cells
    headerEl.innerHTML = weekDays.map((d, idx) => {
      const iso = formatDateISO(d);
      const isToday = iso === todayStr;
      return `
        <div class="week-day-header-cell ${isToday ? 'today' : ''}">
          <div class="w-day-name">${dayNames[idx]}</div>
          <div class="w-day-date">${d.getDate()}.${d.getMonth() + 1}.</div>
        </div>
      `;
    }).join('');

    // Group lessons by date
    const lessonGroups = {};
    weekDays.forEach(d => lessonGroups[formatDateISO(d)] = []);
    lessons.forEach(l => {
      const d = l.date || l.Date;
      if (lessonGroups[d]) {
        lessonGroups[d].push(l);
      }
    });

    // Body columns
    bodyEl.innerHTML = weekDays.map(d => {
      const iso = formatDateISO(d);
      const dayLessons = lessonGroups[iso] || [];

      const boxesHtml = dayLessons.map(l => {
        const color = l.color || '#ff7a00';
        return `
          <div class="week-lesson-box" style="border-left: 4px solid ${color};" onclick="openLessonDetailModal(${escapeHTML(JSON.stringify(l))})">
            <span class="w-time">${l.startTimeStr} - ${l.endTimeStr}</span>
            <div class="w-subj">${l.subject}</div>
            <div class="w-details">${l.room || ''} · ${l.teacher || ''}</div>
          </div>
        `;
      }).join('');

      return `
        <div class="week-day-col">
          ${boxesHtml || '<div class="empty-inline-state" style="padding:12px 0;">Frei</div>'}
        </div>
      `;
    }).join('');
  }

  // ==================== WEITERE STUNDENPLÄNE MODULE ====================
  function getFavoriteClasses() {
    try {
      return JSON.parse(localStorage.getItem('untis_fav_classes') || '[]');
    } catch (e) {
      return [];
    }
  }

  window.toggleFavoriteClass = function(id, e) {
    if (e) e.stopPropagation();
    let favs = getFavoriteClasses();
    const idx = favs.indexOf(id);
    if (idx >= 0) {
      favs.splice(idx, 1);
    } else {
      favs.push(id);
    }
    localStorage.setItem('untis_fav_classes', JSON.stringify(favs));
    renderResourceItems(state.classes, 'CLASS');
  };

  async function loadOtherTimetablesTab() {
    const listEl = document.getElementById('resourceItemsList');
    if (!listEl) return;

    if (state.otherTab !== 'CLASS' && state.otherTab !== 'ROOM') {
      state.otherTab = 'CLASS';
    }

    if (state.otherTab === 'CLASS') {
      if (state.classes.length === 0) {
        showLoading(true, 'Lade Klassen...');
        const classes = await apiFetch('/api/classes');
        state.classes = classes || [];
        showLoading(false);
      }
      renderResourceItems(state.classes, 'CLASS');
    } else if (state.otherTab === 'ROOM') {
      if (state.rooms.length === 0) {
        showLoading(true, 'Lade Fachräume...');
        const rooms = await apiFetch('/api/rooms');
        state.rooms = rooms || [];
        showLoading(false);
      }
      renderResourceItems(state.rooms, 'ROOM');
    }

    if (state.otherSelectedId) {
      loadResourceTimetable();
    }
  }

  function switchResourceTab(tabName) {
    if (tabName !== 'CLASS' && tabName !== 'ROOM') {
      tabName = 'CLASS';
    }
    state.otherTab = tabName;
    document.querySelectorAll('.res-tab-btn').forEach(b => b.classList.remove('active'));
    if (tabName === 'CLASS') document.getElementById('tabResClasses')?.classList.add('active');
    if (tabName === 'ROOM') document.getElementById('tabResRooms')?.classList.add('active');

    const pill = document.getElementById('otherResTypePill');
    if (pill) pill.textContent = tabName === 'CLASS' ? 'KLASSE' : 'FACHRAUM';

    loadOtherTimetablesTab();
  }

  function renderResourceItems(items, type) {
    const listEl = document.getElementById('resourceItemsList');
    if (!listEl) return;

    const search = document.getElementById('resourceSearchInput')?.value.toLowerCase().trim() || '';

    let filtered = items.filter(it => {
      const name = (it.name || it.longName || '').toLowerCase();
      const longName = (it.longName || '').toLowerCase();
      return name.includes(search) || longName.includes(search);
    });

    // Pinned Favorites to top for Klassen
    const favs = getFavoriteClasses();
    if (type === 'CLASS') {
      filtered.sort((a, b) => {
        const aFav = favs.includes(a.id);
        const bFav = favs.includes(b.id);
        if (aFav && !bFav) return -1;
        if (!aFav && bFav) return 1;
        return (a.name || '').localeCompare(b.name || '');
      });
    }

    if (filtered.length === 0) {
      listEl.innerHTML = '<div class="empty-inline-state">Keine Einträge gefunden.</div>';
      return;
    }

    listEl.innerHTML = filtered.map(it => {
      const isSelected = state.otherSelectedId === it.id && state.otherTab === type;
      let label = it.name;
      if (type === 'ROOM' && it.longName && it.longName !== it.name) {
        label = `${it.name} - ${it.longName}`;
      }

      const isFav = favs.includes(it.id);
      const favHtml = type === 'CLASS'
        ? `<button class="btn-fav-heart ${isFav ? 'active' : ''}" onclick="toggleFavoriteClass(${it.id}, event)" title="${isFav ? 'Aus Favoriten entfernen' : 'Zu Favoriten hinzufügen'}">${isFav ? '❤️' : '🤍'}</button>`
        : '';

      return `
        <div class="res-item-row ${isSelected ? 'active' : ''}" onclick="selectResourceItem(${it.id}, '${escapeHTML(label)}', '${type}')">
          <span>${escapeHTML(label)}</span>
          ${favHtml}
        </div>
      `;
    }).join('');
  }

  function selectResourceItem(id, name, type) {
    state.otherSelectedId = id;
    state.otherSelectedName = name;
    state.otherTab = type;

    const heading = document.getElementById('otherResNameHeading');
    if (heading) heading.textContent = name;

    renderResourceItems(
      type === 'CLASS' ? state.classes : state.rooms,
      type
    );

    loadResourceTimetable();
  }

  async function loadResourceTimetable() {
    if (!state.otherSelectedId) return;

    showLoading(true, `Lade Stundenplan für ${state.otherSelectedName}...`);
    const dateStr = formatDateISO(state.currentDate);
    const view = state.timetableMode;

    const lessons = await apiFetch(`/api/timetable/resource?type=${state.otherTab}&id=${state.otherSelectedId}&date=${dateStr}&view=${view}`);
    showLoading(false);

    const dayContainer = document.getElementById('otherDayViewContainer');
    const weekContainer = document.getElementById('otherWeekViewContainer');
    const emptyState = document.getElementById('otherEmptyState');

    if (!lessons || lessons.length === 0) {
      if (dayContainer) dayContainer.style.display = 'none';
      if (weekContainer) weekContainer.style.display = 'none';
      if (emptyState) emptyState.style.display = 'block';
      return;
    }

    if (emptyState) emptyState.style.display = 'none';

    if (view === 'day') {
      if (dayContainer) dayContainer.style.display = 'block';
      if (weekContainer) weekContainer.style.display = 'none';
      renderDayTimetable(lessons, 'otherDayLessonsList');
    } else {
      if (dayContainer) dayContainer.style.display = 'none';
      if (weekContainer) weekContainer.style.display = 'block';
      renderWeekTimetable(lessons, 'otherWeekGridHeader', 'otherWeekGridBody');
    }
  }

  // ==================== HAUSAUFGABEN MODULE ====================
  async function loadHomework() {
    showLoading(true, 'Lade Hausaufgaben...');
    const data = await apiFetch('/api/homework');
    showLoading(false);

    state.homework = data || [];
    renderHomeworkCards();
  }

  function filterHomework(filterMode) {
    state.hwFilter = filterMode;
    document.querySelectorAll('.filter-pills-bar .pill-btn').forEach(b => b.classList.remove('active'));
    if (filterMode === 'all') document.getElementById('hwFilterAll')?.classList.add('active');
    if (filterMode === 'open') document.getElementById('hwFilterOpen')?.classList.add('active');
    if (filterMode === 'completed') document.getElementById('hwFilterCompleted')?.classList.add('active');

    renderHomeworkCards();
  }

  function renderHomeworkCards() {
    const container = document.getElementById('homeworkCardsContainer');
    if (!container) return;

    let items = state.homework;
    if (state.hwFilter === 'open') items = items.filter(h => !h.completed);
    if (state.hwFilter === 'completed') items = items.filter(h => h.completed);

    if (items.length === 0) {
      container.innerHTML = '<div class="empty-state"><h3>Keine Aufgaben gefunden</h3><p>Trage über den Button oben eine neue Hausaufgabe ein.</p></div>';
      return;
    }

    container.innerHTML = items.map(h => {
      const isLocal = h.source !== 'webuntis';
      return `
        <div class="homework-card ${h.completed ? 'completed' : ''}">
          <div class="custom-checkbox ${h.completed ? 'checked' : ''}" onclick="toggleHomeworkComplete('${h.id}', ${!h.completed})">
            ${h.completed ? '✓' : ''}
          </div>
          <div class="hw-info">
            <div class="hw-header-line">
              <span class="hw-subject-badge">${escapeHTML(h.subject)}</span>
              <span class="hw-source-badge">${isLocal ? 'Persönlich' : 'WebUntis'}</span>
              <span class="hw-due-pill">${h.dueDate ? `Fällig: ${h.dueDate}` : ''}</span>
            </div>
            <div class="hw-desc">${escapeHTML(h.description)}</div>
          </div>
          ${isLocal ? `
            <button class="btn-delete-item" title="Aufgabe löschen" onclick="deleteHomeworkItem('${h.id}')">
              <svg viewBox="0 0 24 24" fill="currentColor"><path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/></svg>
            </button>
          ` : ''}
        </div>
      `;
    }).join('');
  }

  async function toggleHomeworkComplete(id, completed) {
    const res = await apiFetch('/api/homework', {
      method: 'PUT',
      body: JSON.stringify({ id, completed }),
    });

    if (res && res.success) {
      const item = state.homework.find(h => h.id === id);
      if (item) item.completed = completed;
      renderHomeworkCards();
      showToast(completed ? 'Aufgabe als erledigt markiert' : 'Aufgabe wieder geöffnet');
    }
  }

  async function deleteHomeworkItem(id) {
    if (!confirm('Möchtest du diese Hausaufgabe wirklich löschen?')) return;

    const res = await apiFetch(`/api/homework?id=${id}`, { method: 'DELETE' });
    if (res && res.success) {
      state.homework = state.homework.filter(h => h.id !== id);
      renderHomeworkCards();
      showToast('Hausaufgabe gelöscht');
    }
  }

  function openNewHomeworkModal() {
    const modal = document.getElementById('homeworkModalBackdrop');
    if (modal) {
      document.getElementById('hwSubjectInput').value = '';
      document.getElementById('hwDescInput').value = '';
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);
      document.getElementById('hwDueDateInput').value = formatDateISO(tomorrow);
      modal.style.display = 'flex';
    }
  }

  function closeNewHomeworkModal() {
    const modal = document.getElementById('homeworkModalBackdrop');
    if (modal) modal.style.display = 'none';
  }

  async function handleCreateHomework(e) {
    e.preventDefault();
    const subject = document.getElementById('hwSubjectInput').value.trim();
    const description = document.getElementById('hwDescInput').value.trim();
    const dueDate = document.getElementById('hwDueDateInput').value;

    if (!description) return;

    const res = await apiFetch('/api/homework', {
      method: 'POST',
      body: JSON.stringify({ subject, description, dueDate }),
    });

    if (res && res.success) {
      closeNewHomeworkModal();
      showToast('Neue Hausaufgabe erfolgreich gespeichert!');
      loadHomework();
    }
  }

  // ==================== ABWESENHEITEN MODULE ====================
  async function loadAbsences() {
    showLoading(true, 'Lade Fehlzeiten...');
    const year = state.selectedSchoolYear || '2026/2027';
    const data = await apiFetch(`/api/absences?year=${encodeURIComponent(year)}&format=full`);
    showLoading(false);

    if (data && Array.isArray(data)) {
      state.absences = data;
    } else if (data && data.absences) {
      state.absences = data.absences;
      if (data.selectedSchoolYear) {
        state.selectedSchoolYear = data.selectedSchoolYear;
        const sel = document.getElementById('absSchoolYearSelect');
        if (sel) sel.value = data.selectedSchoolYear;
      }
    } else {
      state.absences = [];
    }
    renderAbsenceCards();
  }

  window.onAbsenceSchoolYearChange = function(val) {
    state.selectedSchoolYear = val;
    loadAbsences();
  };

  function renderAbsenceCards() {
    const container = document.getElementById('absencesCardsContainer');
    const totalEl = document.getElementById('absTotalCount');
    const excEl = document.getElementById('absExcusedCount');
    const unexcEl = document.getElementById('absUnexcusedCount');

    let total = state.absences.length;
    let excused = 0;
    let unexcused = 0;

    state.absences.forEach(a => {
      if (a.isExcused) excused++;
      else unexcused++;
    });

    if (totalEl) totalEl.textContent = total;
    if (excEl) excEl.textContent = excused;
    if (unexcEl) unexcEl.textContent = unexcused;

    if (!container) return;

    if (state.absences.length === 0) {
      container.innerHTML = '<div class="empty-state"><h3>Keine Fehlzeiten im gewählten Schuljahr</h3><p>Reiche über den Button oben eine Krankmeldung oder Abwesenheit ein.</p></div>';
      return;
    }

    container.innerHTML = state.absences.map(a => {
      const isLocal = a.source !== 'webuntis';
      const statusPill = a.isExcused
        ? '<span class="abs-status-tag excused">Entschuldigt</span>'
        : '<span class="abs-status-tag unexcused">Unentschuldigt</span>';

      return `
        <div class="absence-card">
          <div class="abs-left">
            <div class="abs-icon-box">
              <svg viewBox="0 0 24 24" fill="currentColor"><path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11z"/></svg>
            </div>
            <div>
              <div class="abs-reason-title">${escapeHTML(a.reason || 'Abwesenheit')}</div>
              <div class="abs-date-text">Zeitraum: <strong>${a.startDate || ''} bis ${a.endDate || ''}</strong></div>
              ${a.text ? `<div class="abs-note-text">${escapeHTML(a.text)}</div>` : ''}
            </div>
          </div>
          <div style="display:flex; align-items:center; gap:12px;">
            ${statusPill}
            ${isLocal ? `
              <button class="btn-delete-item" title="Eintrag löschen" onclick="deleteAbsenceItem('${a.id}')">
                <svg viewBox="0 0 24 24" fill="currentColor"><path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/></svg>
              </button>
            ` : ''}
          </div>
        </div>
      `;
    }).join('');
  }

  async function deleteAbsenceItem(id) {
    if (!confirm('Möchtest du diese Abwesenheitsmeldung wirklich löschen?')) return;

    const res = await apiFetch(`/api/absences?id=${id}`, { method: 'DELETE' });
    if (res && res.success) {
      state.absences = state.absences.filter(a => a.id !== id);
      renderAbsenceCards();
      showToast('Abwesenheitseintrag gelöscht');
    }
  }

  function openNewAbsenceModal() {
    const modal = document.getElementById('absenceModalBackdrop');
    if (modal) {
      document.getElementById('absTextInput').value = '';
      const today = formatDateISO(new Date());
      document.getElementById('absStartDateInput').value = today;
      document.getElementById('absEndDateInput').value = today;
      document.getElementById('absExcusedCheckbox').checked = true;
      modal.style.display = 'flex';
    }
  }

  function closeNewAbsenceModal() {
    const modal = document.getElementById('absenceModalBackdrop');
    if (modal) modal.style.display = 'none';
  }

  async function handleCreateAbsence(e) {
    e.preventDefault();
    const reason = document.getElementById('absReasonInput').value;
    const text = document.getElementById('absTextInput').value.trim();
    const startDate = document.getElementById('absStartDateInput').value;
    const endDate = document.getElementById('absEndDateInput').value;
    const isExcused = document.getElementById('absExcusedCheckbox').checked;

    const res = await apiFetch('/api/absences', {
      method: 'POST',
      body: JSON.stringify({ reason, text, startDate, endDate, isExcused }),
    });

    if (res && res.success) {
      closeNewAbsenceModal();
      showToast('Abwesenheitsmeldung erfolgreich hinterlegt!');
      loadAbsences();
    }
  }

  // ==================== MITTEILUNGEN MODULE ====================
  function getReadMessageIds() {
    try {
      return JSON.parse(localStorage.getItem('untis_read_msgs') || '[]');
    } catch(e) {
      return [];
    }
  }

  function markMessageAsRead(id) {
    const ids = getReadMessageIds();
    const idStr = String(id);
    if (!ids.includes(idStr)) {
      ids.push(idStr);
      localStorage.setItem('untis_read_msgs', JSON.stringify(ids));
    }
    updateMessagesBadge();
  }

  window.markAllMessagesAsRead = function() {
    const ids = getReadMessageIds();
    (state.messages || []).forEach(m => {
      const s = String(m.id);
      if (!ids.includes(s)) ids.push(s);
    });
    localStorage.setItem('untis_read_msgs', JSON.stringify(ids));
    updateMessagesBadge();
    renderMessages();
    showToast('Alle Mitteilungen als gelesen markiert.');
  };

  function updateMessagesBadge() {
    const readIds = getReadMessageIds();
    const unreadCount = (state.messages || []).filter(m => !readIds.includes(String(m.id))).length;
    const msgBadge = document.getElementById('sidebarMsgBadge');
    if (msgBadge) {
      if (unreadCount > 0) {
        msgBadge.textContent = unreadCount;
        msgBadge.style.display = 'inline-block';
      } else {
        msgBadge.textContent = '0';
        msgBadge.style.display = 'none';
      }
    }
    const dashMsgCount = document.getElementById('dashMsgCount');
    const dashMsgHeadline = document.getElementById('dashMsgHeadline');
    if (dashMsgCount) dashMsgCount.textContent = unreadCount > 0 ? `${unreadCount} ungelesen` : '0 ungelesen';
    if (dashMsgHeadline) dashMsgHeadline.textContent = unreadCount > 0 ? `${unreadCount} neue Mitteilungen` : 'Alle Mitteilungen gelesen';
  }

  async function loadMessages() {
    showLoading(true, 'Lade Mitteilungen...');
    const data = await apiFetch('/api/messages');
    showLoading(false);

    state.messages = data || [];
    updateMessagesBadge();
    renderMessages();
  }

  function renderMessages() {
    const container = document.getElementById('messagesContainer');
    if (!container) return;

    if (state.messages.length === 0) {
      container.innerHTML = '<div class="empty-state"><h3>Keine Mitteilungen vorhanden</h3><p>Derzeit liegen keine Durchsagen oder Nachrichten vor.</p></div>';
      return;
    }

    const readIds = getReadMessageIds();

    container.innerHTML = state.messages.map(m => {
      const isUnread = !readIds.includes(String(m.id));
      return `
        <div class="message-card ${isUnread ? 'unread' : ''}" onclick="openMessageDetailModal(${m.id})">
          <div class="msg-header-row">
            <div style="display:flex; align-items:center;">
              ${isUnread ? '<span class="msg-unread-tag" title="Ungelesen"></span>' : ''}
              <span class="msg-sender-chip">${m.senderName || 'Schule'}</span>
            </div>
            <span class="msg-date-str">${formatDateTime(m.sentDateTime)}</span>
          </div>
          <div class="msg-subject-line">${escapeHTML(m.subject)}</div>
          <div class="msg-preview-text">${escapeHTML(m.contentPreview || '')}</div>
        </div>
      `;
    }).join('');
  }

  async function openMessageDetailModal(id) {
    const modal = document.getElementById('messageDetailBackdrop');
    if (!modal) return;

    markMessageAsRead(id);
    renderMessages();

    modal.style.display = 'flex';
    document.getElementById('msgDetailSubject').textContent = 'Lade Nachricht...';
    document.getElementById('msgDetailContent').textContent = '';

    const msg = await apiFetch(`/api/messages/${id}`);
    if (msg) {
      document.getElementById('msgDetailSender').textContent = msg.senderName || (msg.sender && msg.sender.displayName) || 'Schule';
      document.getElementById('msgDetailDate').textContent = formatDateTime(msg.sentDateTime);
      document.getElementById('msgDetailSubject').textContent = msg.subject;
      document.getElementById('msgDetailContent').textContent = msg.content || msg.contentPreview || 'Kein Textinhalt verfügbar.';
    }
  }

  function closeMessageDetailModal() {
    const modal = document.getElementById('messageDetailBackdrop');
    if (modal) modal.style.display = 'none';
  }

  // ==================== PROFILE & SCHULEN (MIT LÖSCHFUNKTION!) ====================
  async function loadProfiles() {
    showLoading(true, 'Lade Profile...');
    const data = await apiFetch('/api/profiles');
    showLoading(false);

    if (data && data.profiles) {
      state.profiles = data.profiles;
      renderProfilesManagement(data.profiles, data.activeProfileId);
    }
  }

  function renderProfilesManagement(profiles, activeId) {
    const grid = document.getElementById('profilesManagementList');
    if (!grid) return;

    if (!profiles || profiles.length === 0) {
      grid.innerHTML = '<div class="empty-inline-state">Keine Profile gespeichert. Füge unten deine Schule hinzu.</div>';
      return;
    }

    grid.innerHTML = profiles.map(p => {
      const isActive = p.id === activeId || p.isActive;
      return `
        <div class="profile-card ${isActive ? 'active' : ''}">
          <div class="prof-top-row">
            <div>
              <div class="prof-name-text">${escapeHTML(p.name)}</div>
              <div class="prof-school-text">${escapeHTML(p.school)}</div>
              <div class="prof-user-text">Benutzer: ${escapeHTML(p.username || 'anonym')}</div>
            </div>
            ${isActive ? '<span class="active-profile-badge">AKTIV</span>' : ''}
          </div>

          <div class="prof-actions-row">
            <div>
              ${!isActive ? `
                <button class="btn-switch-profile" onclick="switchActiveProfile('${p.id}')">Als aktiv wählen</button>
              ` : '<span style="font-size:12px; color:var(--text-muted);">Aktive Verbindung</span>'}
            </div>

            <!-- DER ROTE PAPIERKORB-BUTTON ZUM LÖSCHEN DER SCHULE / DES PROFILS! -->
            <button class="btn-delete-profile" title="Schule / Profil löschen" onclick="deleteProfile('${p.id}', '${escapeHTML(p.name)}')">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
              </svg>
            </button>
          </div>
        </div>
      `;
    }).join('');
  }

  async function switchActiveProfile(profileId) {
    showLoading(true, 'Wechsle Profil...');
    const res = await apiFetch('/api/profiles/switch', {
      method: 'POST',
      body: JSON.stringify({ profileId }),
    });
    showLoading(false);

    if (res && res.success) {
      showToast(res.message || 'Profil gewechselt!');
      // Reload app
      window.location.reload();
    } else {
      showToast(res?.message || 'Fehler beim Wechseln des Profils', 'error');
    }
  }

  // PROFIL LÖSCHEN (ROTER PAPIERKORB)
  async function deleteProfile(profileId, profileName) {
    const confirmDelete = confirm(`Möchtest du das Profil '${profileName}' wirklich löschen?\n\nAlle lokalen Hausaufgaben und Abwesenheiten für dieses Profil werden entfernt.`);
    if (!confirmDelete) return;

    showLoading(true, 'Lösche Profil...');
    const res = await apiFetch(`/api/profiles/delete?id=${encodeURIComponent(profileId)}`, {
      method: 'POST',
    });
    showLoading(false);

    if (res && res.success) {
      showToast(`Profil '${profileName}' wurde gelöscht!`);
      // Reload profile list
      loadProfiles();
      // Check if we need to reload app state
      const status = await apiFetch('/api/status');
      if (status) {
        state.status = status;
        if (status.needsOnboarding) {
          window.location.reload();
        }
      }
    } else {
      showToast(res?.message || 'Fehler beim Löschen des Profils', 'error');
    }
  }

  // School Search & Add
  let selectedSearchSchool = null;

  async function handleSchoolSearch() {
    const query = document.getElementById('addSchoolSearchInput')?.value.trim();
    if (!query) return;

    const list = document.getElementById('addSchoolResultsList');
    if (list) list.innerHTML = '<div class="empty-inline-state">Suche Schulen...</div>';

    const data = await apiFetch(`/api/schools/search?q=${encodeURIComponent(query)}`);
    if (!list) return;

    if (!data || !data.schools || data.schools.length === 0) {
      list.innerHTML = '<div class="empty-inline-state">Keine Schulen gefunden. Bitte Suchbegriff anpassen.</div>';
      return;
    }

    list.innerHTML = data.schools.map(s => `
      <div class="school-result-item" onclick="selectSearchSchool(${escapeHTML(JSON.stringify(s))})">
        <div>
          <div class="sr-name">${escapeHTML(s.displayName)}</div>
          <div class="sr-details">${escapeHTML(s.address || s.serverUrl || s.server)}</div>
        </div>
        <button class="btn-text-sm">Auswählen &rarr;</button>
      </div>
    `).join('');
  }

  function selectSearchSchool(school) {
    selectedSearchSchool = school;
    const form = document.getElementById('addSchoolForm');
    const preview = document.getElementById('selectedSchoolPreview');

    if (preview) {
      preview.textContent = `Ausgewählte Schule: ${school.displayName} (${school.loginName || school.server})`;
    }
    if (form) {
      form.style.display = 'flex';
      form.scrollIntoView({ behavior: 'smooth' });
    }
  }

  function cancelAddSchool() {
    selectedSearchSchool = null;
    const form = document.getElementById('addSchoolForm');
    if (form) form.style.display = 'none';
  }

  async function handleSaveNewSchool(e) {
    e.preventDefault();
    if (!selectedSearchSchool) return;

    const username = document.getElementById('newUsername').value.trim();
    const password = document.getElementById('newPassword').value;
    const profileName = document.getElementById('newProfileName').value.trim();

    showLoading(true, 'Verbinde mit WebUntis...');
    const res = await apiFetch('/api/profiles', {
      method: 'POST',
      body: JSON.stringify({
        school: selectedSearchSchool.loginName || selectedSearchSchool.displayName,
        server: selectedSearchSchool.serverUrl || selectedSearchSchool.server,
        name: profileName || selectedSearchSchool.displayName,
        username,
        password,
        setActive: true,
      }),
    });
    showLoading(false);

    if (res && res.success) {
      showToast('Erfolgreich angemeldet und Schule hinzugefügt!');
      window.location.reload();
    } else {
      showToast(res?.message || 'Anmeldung fehlgeschlagen. Bitte Zugangsdaten prüfen.', 'error');
    }
  }

  function openProfilesModal() {
    const modal = document.getElementById('profilesModalBackdrop');
    if (modal) {
      modal.style.display = 'flex';
      loadProfiles();
    }
  }

  function closeProfilesModal() {
    const modal = document.getElementById('profilesModalBackdrop');
    if (modal) modal.style.display = 'none';
    cancelAddSchool();
  }

  // ==================== UPDATE SYSTEM MODULE ====================
  let availableUpdateInfo = null;

  async function checkForSoftwareUpdates(silent = false) {
    try {
      const res = await apiFetch('/api/updates/check');
      if (res && res.hasUpdate) {
        availableUpdateInfo = res;
        const topBadge = document.getElementById('topbarUpdateBadge');
        const topText = document.getElementById('topbarUpdateText');
        if (topBadge) topBadge.style.display = 'inline-flex';
        if (topText) topText.textContent = `Update ${res.latestVersion} verfügbar`;

        if (!silent) {
          openUpdateModal();
        }
      } else {
        if (!silent) {
          showToast(`Untis Desktop ist auf dem neuesten Stand (${res?.currentVersion || 'aktuell'}).`);
        }
      }
    } catch (e) {
      if (!silent) {
        showToast('Fehler beim Prüfen auf Updates', 'error');
      }
    }
  }

  function openUpdateModal() {
    if (!availableUpdateInfo) return;
    const modal = document.getElementById('updateModalBackdrop');
    if (!modal) return;

    const currVerEl = document.getElementById('updateCurrentVer');
    const newVerEl = document.getElementById('updateNewVer');
    const titleEl = document.getElementById('updateReleaseTitle');
    const notesEl = document.getElementById('updateReleaseNotes');
    const progWrap = document.getElementById('updateProgressWrap');
    const btn = document.getElementById('btnApplyUpdate');

    if (currVerEl) currVerEl.textContent = availableUpdateInfo.currentVersion || 'v1.0.0';
    if (newVerEl) newVerEl.textContent = availableUpdateInfo.latestVersion || 'v1.1.0';
    if (titleEl) titleEl.textContent = availableUpdateInfo.title || `Untis Desktop ${availableUpdateInfo.latestVersion}`;
    if (notesEl) notesEl.textContent = availableUpdateInfo.releaseNotes || 'Keine Versionshinweise verfügbar.';

    if (progWrap) progWrap.style.display = 'none';
    if (btn) {
      btn.disabled = false;
      btn.textContent = 'Jetzt aktualisieren';
    }

    modal.style.display = 'flex';
  }

  function closeUpdateModal() {
    const modal = document.getElementById('updateModalBackdrop');
    if (modal) modal.style.display = 'none';
  }

  async function applySoftwareUpdate() {
    if (!availableUpdateInfo) return;

    const btn = document.getElementById('btnApplyUpdate');
    const progressWrap = document.getElementById('updateProgressWrap');
    const progressText = document.getElementById('updateProgressText');

    if (btn) btn.disabled = true;
    if (progressWrap) progressWrap.style.display = 'block';
    if (progressText) progressText.textContent = 'Update wird heruntergeladen und installiert...';

    try {
      const res = await apiFetch('/api/updates/apply', {
        method: 'POST',
        body: JSON.stringify({ downloadUrl: availableUpdateInfo.downloadUrl }),
      });

      if (res && res.success) {
        if (progressText) progressText.textContent = 'Update erfolgreich installiert! Starte App neu...';
        showToast('Update erfolgreich installiert!');
        setTimeout(() => {
          window.location.reload();
        }, 2000);
      } else {
        if (btn) btn.disabled = false;
        if (progressText) progressText.textContent = `Fehler: ${res?.message || 'Update fehlgeschlagen'}`;
        showToast(res?.message || 'Update fehlgeschlagen', 'error');
      }
    } catch (err) {
      if (btn) btn.disabled = false;
      if (progressText) progressText.textContent = 'Fehler beim Ausführen des Updates.';
      showToast('Fehler beim Ausführen des Updates', 'error');
    }
  }

  // ==================== LESSON DETAIL MODAL ====================
  function openLessonDetailModal(lesson) {
    const modal = document.getElementById('lessonDetailBackdrop');
    if (!modal) return;

    const chip = document.getElementById('lessonModalSubjChip');
    if (chip) {
      chip.textContent = (lesson.subject || 'LF').slice(0, 4);
      chip.style.backgroundColor = lesson.color || '#ff7a00';
      chip.style.color = lesson.textColor || '#ffffff';
    }

    document.getElementById('lessonModalSubject').textContent = lesson.subjectLong || lesson.subject;
    document.getElementById('lessonModalTime').textContent = `${lesson.timeRange} (${lesson.period})`;
    document.getElementById('lessonModalTeacher').textContent = lesson.teacherLong || lesson.teacher || '-';
    document.getElementById('lessonModalRoom').textContent = lesson.roomLong || lesson.room || '-';
    document.getElementById('lessonModalClass').textContent = lesson.class || '-';

    let statusText = 'Regulär';
    if (lesson.isCancelled) statusText = 'ENTFALLEN';
    else if (lesson.isSubstitution) statusText = `Vertretung (${lesson.substText || 'Änderung'})`;
    else if (lesson.isRoomChange) statusText = 'Raumänderung';
    document.getElementById('lessonModalStatus').textContent = statusText;

    const tcWrap = document.getElementById('lessonModalTeachingContentWrap');
    if (tcWrap) {
      if (lesson.teachingContent) {
        tcWrap.style.display = 'block';
        document.getElementById('lessonModalTeachingContent').textContent = lesson.teachingContent;
      } else {
        tcWrap.style.display = 'none';
      }
    }

    const hwWrap = document.getElementById('lessonModalHomeworkWrap');
    if (hwWrap) {
      if (lesson.homeworks && lesson.homeworks.length > 0) {
        hwWrap.style.display = 'block';
        document.getElementById('lessonModalHomework').textContent = lesson.homeworks.join('\n');
      } else {
        hwWrap.style.display = 'none';
      }
    }

    const notesWrap = document.getElementById('lessonModalNotesWrap');
    if (notesWrap) {
      if (lesson.notes || lesson.substText) {
        notesWrap.style.display = 'block';
        document.getElementById('lessonModalNotes').textContent = lesson.notes || lesson.substText;
      } else {
        notesWrap.style.display = 'none';
      }
    }

    modal.style.display = 'flex';
  }

  function closeLessonDetailModal() {
    const modal = document.getElementById('lessonDetailBackdrop');
    if (modal) modal.style.display = 'none';
  }

  // ==================== DATE & LIVE MARKER ====================
  function updateDateDisplay() {
    const label = document.getElementById('dateNavLabel');
    if (label) {
      if (state.timetableMode === 'day') {
        label.textContent = formatGermanDate(state.currentDate);
      } else {
        // Week range
        const d = new Date(state.currentDate);
        const dayOfWeek = d.getDay();
        const diff = d.getDate() - dayOfWeek + (dayOfWeek === 0 ? -6 : 1);
        const mon = new Date(d.setDate(diff));
        const fri = new Date(mon);
        fri.setDate(mon.getDate() + 4);
        label.textContent = `${mon.getDate()}.${mon.getMonth() + 1}. - ${fri.getDate()}.${fri.getMonth() + 1}.${fri.getFullYear()}`;
      }
    }
  }

  function navigateDate(direction) {
    const d = new Date(state.currentDate);
    const amount = state.timetableMode === 'week' ? 7 : 1;
    d.setDate(d.getDate() + (direction * amount));
    state.currentDate = d;
    updateDateDisplay();

    if (state.currentView === 'own-timetable') {
      loadOwnTimetable();
    } else if (state.currentView === 'other-timetables') {
      loadResourceTimetable();
    }
  }

  function jumpToToday() {
    state.currentDate = new Date();
    updateDateDisplay();

    if (state.currentView === 'own-timetable') {
      loadOwnTimetable();
    } else if (state.currentView === 'other-timetables') {
      loadResourceTimetable();
    }
  }

  function setTimetableMode(mode) {
    state.timetableMode = mode;
    document.getElementById('segViewDay')?.classList.toggle('active', mode === 'day');
    document.getElementById('segViewWeek')?.classList.toggle('active', mode === 'week');
    updateDateDisplay();

    if (state.currentView === 'own-timetable') {
      loadOwnTimetable();
    } else if (state.currentView === 'other-timetables') {
      loadResourceTimetable();
    }
  }

  function updateLiveTimeMarker() {
    const marker = document.getElementById('liveTimelineMarker');
    if (!marker) return;

    const isToday = formatDateISO(state.currentDate) === formatDateISO(new Date());
    if (!isToday || state.timetableMode !== 'day') {
      marker.style.display = 'none';
      return;
    }

    const now = new Date();
    const hours = now.getHours();
    const minutes = now.getMinutes();

    // Start 07:30 to 16:30
    const startMin = 7 * 60 + 30; // 450
    const endMin = 16 * 60 + 30;  // 990
    const currentMin = hours * 60 + minutes;

    if (currentMin >= startMin && currentMin <= endMin) {
      marker.style.display = 'flex';
      const label = document.getElementById('liveTimelineTimeLabel');
      if (label) label.textContent = `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}`;
    } else {
      marker.style.display = 'none';
    }
  }

  // ==================== THEME & EVENT LISTENERS ====================
  function setupTheme() {
    const savedTheme = localStorage.getItem('untis_theme') || 'dark';
    document.documentElement.setAttribute('data-theme', savedTheme);
  }

  function toggleTheme() {
    const current = document.documentElement.getAttribute('data-theme');
    const next = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('untis_theme', next);
    showToast(`Farbschema geändert zu: ${next === 'dark' ? 'Dunkel' : 'Hell'}`);
  }

  function setupEventListeners() {
    // Nav Buttons
    document.getElementById('navDashboard')?.addEventListener('click', () => switchView('dashboard'));
    document.getElementById('navOwnTimetable')?.addEventListener('click', () => switchView('own-timetable'));
    document.getElementById('navOtherTimetables')?.addEventListener('click', () => switchView('other-timetables'));
    document.getElementById('navHomework')?.addEventListener('click', () => switchView('homework'));
    document.getElementById('navAbsences')?.addEventListener('click', () => switchView('absences'));
    document.getElementById('navMessages')?.addEventListener('click', () => switchView('messages'));
    document.getElementById('navProfiles')?.addEventListener('click', () => switchView('profiles'));

    // Sidebar footer
    document.getElementById('sidebarThemeBtn')?.addEventListener('click', toggleTheme);
    document.getElementById('sidebarRefreshBtn')?.addEventListener('click', async () => {
      showLoading(true, 'Synchronisiere mit WebUntis...');
      await apiFetch('/api/refresh', { method: 'POST' });
      showLoading(false);
      showToast('Cache geleert und Daten synchronisiert!');
      switchView(state.currentView);
    });

    // Date navigation
    document.getElementById('dateNavPrev')?.addEventListener('click', () => navigateDate(-1));
    document.getElementById('dateNavNext')?.addEventListener('click', () => navigateDate(1));
    document.getElementById('dateNavToday')?.addEventListener('click', jumpToToday);
    document.getElementById('ownJumpToNextBtn')?.addEventListener('click', () => navigateDate(1));

    // Date picker button
    const dateBtn = document.getElementById('datePickerBtn');
    const nativeDate = document.getElementById('nativeDateInput');
    if (dateBtn && nativeDate) {
      dateBtn.addEventListener('click', () => {
        nativeDate.showPicker ? nativeDate.showPicker() : nativeDate.focus();
      });
      nativeDate.addEventListener('change', (e) => {
        if (e.target.value) {
          state.currentDate = new Date(e.target.value);
          updateDateDisplay();
          if (state.currentView === 'own-timetable') loadOwnTimetable();
          else if (state.currentView === 'other-timetables') loadResourceTimetable();
        }
      });
    }

    // Segmented toggle
    document.getElementById('segViewDay')?.addEventListener('click', () => setTimetableMode('day'));
    document.getElementById('segViewWeek')?.addEventListener('click', () => setTimetableMode('week'));

    // Search input in Other Timetables
    document.getElementById('resourceSearchInput')?.addEventListener('input', () => {
      renderResourceItems(
        state.otherTab === 'CLASS' ? state.classes : (state.otherTab === 'TEACHER' ? state.teachers : state.rooms),
        state.otherTab
      );
    });

    // Homework modal
    document.getElementById('formNewHomework')?.addEventListener('submit', handleCreateHomework);

    // Absence modal
    document.getElementById('formNewAbsence')?.addEventListener('submit', handleCreateAbsence);

    // School search
    document.getElementById('btnSearchSchool')?.addEventListener('click', handleSchoolSearch);
    document.getElementById('addSchoolSearchInput')?.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') handleSchoolSearch();
    });
    document.getElementById('addSchoolForm')?.addEventListener('submit', handleSaveNewSchool);

    // Global keyboard shortcuts
    window.addEventListener('keydown', (e) => {
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

      if (e.key === 'Escape') {
        closeProfilesModal();
        closeUpdateModal();
        closeNewHomeworkModal();
        closeNewAbsenceModal();
        closeMessageDetailModal();
        closeLessonDetailModal();
      } else if (e.key === 'ArrowLeft') {
        navigateDate(-1);
      } else if (e.key === 'ArrowRight') {
        navigateDate(1);
      } else if (e.key.toLowerCase() === 't') {
        jumpToToday();
      } else if (e.key.toLowerCase() === 'd') {
        setTimetableMode('day');
      } else if (e.key.toLowerCase() === 'w') {
        setTimetableMode('week');
      }
    });

    // Live marker tick
    setInterval(updateLiveTimeMarker, 30000);
  }

  // Utilities
  function escapeHTML(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  function formatDateTime(dtStr) {
    if (!dtStr) return '';
    try {
      const d = new Date(dtStr);
      return `${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
    } catch {
      return dtStr;
    }
  }

  // Expose global methods for inline HTML onclick handlers
  window.switchView = switchView;
  window.switchResourceTab = switchResourceTab;
  window.selectResourceItem = selectResourceItem;
  window.filterHomework = filterHomework;
  window.toggleHomeworkComplete = toggleHomeworkComplete;
  window.deleteHomeworkItem = deleteHomeworkItem;
  window.openNewHomeworkModal = openNewHomeworkModal;
  window.closeNewHomeworkModal = closeNewHomeworkModal;
  window.deleteAbsenceItem = deleteAbsenceItem;
  window.openNewAbsenceModal = openNewAbsenceModal;
  window.closeNewAbsenceModal = closeNewAbsenceModal;
  window.loadMessages = loadMessages;
  window.openMessageDetailModal = openMessageDetailModal;
  window.closeMessageDetailModal = closeMessageDetailModal;
  window.openLessonDetailModal = openLessonDetailModal;
  window.closeLessonDetailModal = closeLessonDetailModal;
  window.switchActiveProfile = switchActiveProfile;
  window.deleteProfile = deleteProfile;
  window.selectSearchSchool = selectSearchSchool;
  window.cancelAddSchool = cancelAddSchool;
  window.openProfilesModal = openProfilesModal;
  window.closeProfilesModal = closeProfilesModal;
  window.checkForSoftwareUpdates = checkForSoftwareUpdates;
  window.openUpdateModal = openUpdateModal;
  window.closeUpdateModal = closeUpdateModal;
  window.applySoftwareUpdate = applySoftwareUpdate;

  // Run app on DOMContentLoaded
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initApp);
  } else {
    initApp();
  }
})();
