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
    subjectAliases: {},
    dashboardData: null,
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

  // Global lesson registry to avoid inline JSON attribute escaping bugs
  window.__lessonStore = new Map();
  function registerLesson(l) {
    if (!l) return '';
    const id = 'les_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now();
    window.__lessonStore.set(id, l);
    return id;
  }
  function openLessonById(id) {
    const lesson = window.__lessonStore.get(id);
    if (lesson) {
      openLessonDetailModal(lesson);
    }
  }
  window.openLessonById = openLessonById;

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

  function getDisplaySubject(item) {
    if (!item) return '';
    const code = typeof item === 'string' ? item : (item.subject || '');
    if (state.subjectAliases && state.subjectAliases[code] && state.subjectAliases[code].trim()) {
      const alias = state.subjectAliases[code].trim();
      return `${alias} · ${code}`;
    }
    return code;
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
      'about': 'viewPaneAbout',
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
      'about': { title: 'Über & Info', desc: 'Versionsinformationen, Neuigkeiten & Mitwirkende' },
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
      case 'about':
        loadAboutView();
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

    // Load custom subject aliases
    await loadSubjectAliases();

    // Automatic update check in background
    checkForSoftwareUpdates(true);

    // Start live clock & period countdown timer
    setInterval(updateLiveClockAndCountdown, 1000);
    updateLiveClockAndCountdown();

    if (status.needsOnboarding) {
      openOnboardingWizard();
      return;
    }

    // Load initial dashboard
    updateDateDisplay();
    switchView('dashboard');
  }

  // ==================== LIVE DIGITAL CLOCK & PERIOD TIMER ====================
  function updateLiveClockAndCountdown() {
    const clockEl = document.getElementById('dashDigitalClock');
    const textEl = document.getElementById('dashCountdownText');
    const dotEl = document.querySelector('.dash-clock-dot');
    if (!clockEl && !textEl) return;

    const now = new Date();
    const h = String(now.getHours()).padStart(2, '0');
    const m = String(now.getMinutes()).padStart(2, '0');
    const s = String(now.getSeconds()).padStart(2, '0');
    if (clockEl) clockEl.textContent = `${h}:${m}:${s}`;

    if (!textEl) return;

    const data = state.dashboardData;
    if (!data) {
      textEl.textContent = 'Bereit';
      return;
    }

    const bigTimerEl = document.getElementById('dashHeroGreeting');
    const eyebrowEl = document.getElementById('dashCountdownEyebrow');

    if (data.isUpcomingSchoolDay && data.nextLesson) {
      if (dotEl) dotEl.className = 'dash-clock-dot free';
      const nextSubj = getDisplaySubject(data.nextLesson);
      const nextRoom = data.nextLesson.room ? ` in ${data.nextLesson.room}` : '';
      const dStr = data.nextLesson.date || data.nextLesson.Date;
      const tStr = data.nextLesson.startTimeStr || '07:30';

      if (dStr) {
        const [yr, mo, dy] = dStr.split('-').map(Number);
        const [th, tm] = tStr.split(':').map(Number);
        const target = new Date(yr, mo - 1, dy, th, tm, 0);
        const diffMs = target - now;

        if (diffMs > 0) {
          const totSec = Math.floor(diffMs / 1000);
          const days = Math.floor(totSec / 86400);
          const hours = Math.floor((totSec % 86400) / 3600);
          const mins = Math.floor((totSec % 3600) / 60);
          const secs = totSec % 60;
          const timePart = `${String(hours).padStart(2, '0')}:${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;

          if (eyebrowEl) eyebrowEl.textContent = `${data.upcomingDayLabel || 'Nächster Schultag'} beginnt in:`;
          if (bigTimerEl) bigTimerEl.textContent = days > 0 ? `${days}d ${timePart}` : timePart;

          if (days > 0) {
            textEl.textContent = `${data.upcomingDayLabel || 'Nächster Schultag'}: ${nextSubj}${nextRoom} in ${days}T ${timePart}`;
          } else {
            textEl.textContent = `${data.upcomingDayLabel || 'Nächste Std'}: ${nextSubj}${nextRoom} in ${timePart}`;
          }
          return;
        }
      }
      if (eyebrowEl) eyebrowEl.textContent = `${data.upcomingDayLabel || 'Nächster Schultag'}:`;
      if (bigTimerEl) bigTimerEl.textContent = tStr;
      textEl.textContent = `${data.upcomingDayLabel || 'Nächster Schultag'}: ${nextSubj}${nextRoom} (${tStr})`;
      return;
    }

    const lessons = data.todayLessons || [];
    const nowSec = now.getHours() * 3600 + now.getMinutes() * 60 + now.getSeconds();
    let activeLesson = null;
    let nextUpcoming = null;

    for (const l of lessons) {
      if (l.isCancelled) continue;
      const [sh, sm] = (l.startTimeStr || '00:00').split(':').map(Number);
      const [eh, em] = (l.endTimeStr || '00:00').split(':').map(Number);
      const startSec = sh * 3600 + sm * 60;
      const endSec = eh * 3600 + em * 60;

      if (nowSec >= startSec && nowSec < endSec) {
        activeLesson = { lesson: l, remainingSec: endSec - nowSec };
        break;
      } else if (startSec > nowSec) {
        if (!nextUpcoming || startSec < nextUpcoming.startSec) {
          nextUpcoming = { lesson: l, waitSec: startSec - nowSec, startSec };
        }
      }
    }

    if (activeLesson) {
      if (dotEl) dotEl.className = 'dash-clock-dot';
      const subj = getDisplaySubject(activeLesson.lesson);
      const room = activeLesson.lesson.room ? ` in ${activeLesson.lesson.room}` : '';
      const mins = Math.floor(activeLesson.remainingSec / 60);
      const secs = activeLesson.remainingSec % 60;
      const timePart = `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
      if (eyebrowEl) eyebrowEl.textContent = `Jetzt: ${subj}${room}`;
      if (bigTimerEl) bigTimerEl.textContent = `Noch ${timePart}`;
      textEl.textContent = `Jetzt: ${subj}${room} · Noch ${timePart} bis Pause`;
    } else if (nextUpcoming) {
      if (dotEl) dotEl.className = 'dash-clock-dot in-break';
      const subj = getDisplaySubject(nextUpcoming.lesson);
      const room = nextUpcoming.lesson.room ? ` in ${nextUpcoming.lesson.room}` : '';
      const mins = Math.floor(nextUpcoming.waitSec / 60);
      const secs = nextUpcoming.waitSec % 60;
      const timePart = `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
      if (eyebrowEl) eyebrowEl.textContent = `Nächste Stunde in:`;
      if (bigTimerEl) bigTimerEl.textContent = timePart;
      textEl.textContent = `Pause · Nächste Std: ${subj}${room} in ${timePart}`;
    } else {
      if (dotEl) dotEl.className = 'dash-clock-dot free';
      if (eyebrowEl) eyebrowEl.textContent = `Schultag beendet:`;
      if (bigTimerEl) bigTimerEl.textContent = `Freizeit 🎉`;
      textEl.textContent = 'Schultag beendet · Schöne Freizeit!';
    }
  }

  // ==================== DASHBOARD MODULE ====================
  async function loadDashboard() {
    const data = await apiFetch('/api/dashboard');
    if (!data) return;
    state.dashboardData = data;

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

    // Mitteilungen: calculate unread count based on read message IDs
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

    // Home Section Badges
    const hwCountBadge = document.getElementById('dashHwCountBadge');
    if (hwCountBadge) hwCountBadge.textContent = data.openHomeworkCount || 0;

    const msgCountBadge = document.getElementById('dashMsgCountBadge');
    if (msgCountBadge) msgCountBadge.textContent = `${unreadCount} neu`;

    const absCountBadge = document.getElementById('dashAbsCountBadge');
    if (absCountBadge) absCountBadge.textContent = `${data.absencesSummary?.total || 0}`;

    const absExcEl = document.getElementById('dashAbsExcused');
    const absUnexcEl = document.getElementById('dashAbsUnexcused');
    if (data.absencesSummary) {
      if (absExcEl) absExcEl.textContent = `${data.absencesSummary.excused} Entschuldigt`;
      if (absUnexcEl) absUnexcEl.textContent = `${data.absencesSummary.unexcused} Unentschuldigt`;
    }

    // Today's Lessons List (or upcoming day)
    renderDashboardLessons(data.todayLessons || [], data.isUpcomingSchoolDay, data.upcomingDayLabel);

    // Homework List
    renderDashboardHomework(data.openHomework || []);

    // Messages List
    renderDashboardMessages(data.recentMessages || []);

    // Live clock update
    updateLiveClockAndCountdown();
  }

  function renderDashboardLessons(lessons, isUpcomingDay, upcomingLabel) {
    const list = document.getElementById('dashTodayLessonsList');
    const subtitleEl = document.getElementById('dashScheduleSubtitle');
    if (subtitleEl) {
      subtitleEl.textContent = isUpcomingDay && upcomingLabel ? upcomingLabel : 'Heute';
    }
    if (!list) return;

    // Deduplicate lessons
    const seen = new Set();
    const uniqueLessons = (lessons || []).filter(l => {
      const key = `${l.date || l.Date}_${l.startTimeStr}_${l.endTimeStr}_${l.subject}_${l.room}_${l.teacher}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });

    if (uniqueLessons.length === 0) {
      list.innerHTML = '<div class="empty-inline-state">Heute steht kein planmäßiger Unterricht an.</div>';
      return;
    }

    const headerNote = isUpcomingDay && upcomingLabel
      ? `<div style="font-size:12px; font-weight:700; color:var(--accent-primary); margin-bottom:10px; display:flex; align-items:center; gap:6px;">
           <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>
           ${escapeHTML(upcomingLabel)}
         </div>`
      : '';

    list.innerHTML = headerNote + uniqueLessons.map(l => {
      const color = l.color || '#ff7a00';
      const isSubst = l.isSubstitution || l.isRoomChange;
      const isCanc = l.isCancelled;
      const displaySubj = getDisplaySubject(l);
      const roomText = l.room || '';
      const teacherText = l.teacher || '';

      const statusBadge = isCanc 
        ? '<span class="status-pill cancelled">Ausfall</span>' 
        : (isSubst ? '<span class="status-pill substitution">Änderung</span>' : '');

      const storeId = registerLesson(l);
      const nextL = state.dashboardData?.nextLesson;
      const isNext = (nextL && 
                      (nextL.date || nextL.Date) === (l.date || l.Date) &&
                      nextL.startTimeStr === l.startTimeStr &&
                      nextL.subject === l.subject);

      return `
        <div class="dash-lesson-item ${isSubst ? 'substitution' : ''} ${isCanc ? 'cancelled' : ''} ${isNext ? 'is-next-lesson' : ''}" onclick="openLessonById('${storeId}')">
          <div class="lesson-time-col">
            <span class="l-time-range">${escapeHTML(l.timeRange)}</span>
            <span class="l-period">${escapeHTML(l.period)}</span>
          </div>
          <div class="lesson-color-bar" style="background-color:${color};"></div>
          <div class="lesson-main-col">
            <div class="l-subject-row">
              <span class="l-subject">${escapeHTML(displaySubj)}</span>
              ${roomText ? `<span class="l-room-tag">Raum ${escapeHTML(roomText)}</span>` : ''}
              ${statusBadge}
            </div>
            <span class="l-teacher">${escapeHTML(teacherText)}</span>
          </div>
          ${teacherText ? `<span class="l-teacher-badge">${escapeHTML(teacherText.slice(0, 4))}</span>` : ''}
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
            <span class="hw-subject-badge">${escapeHTML(h.subject)}</span>
            <span class="hw-due-pill">${h.dueDate ? `Fällig: ${escapeHTML(h.dueDate)}` : ''}</span>
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
          <span class="msg-sender-chip">${escapeHTML(m.senderName || 'Schule')}</span>
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

    // Deduplicate lessons
    const seen = new Set();
    const uniqueLessons = (lessons || []).filter(l => {
      const key = `${l.date || l.Date}_${l.startTimeStr}_${l.endTimeStr}_${l.subject}_${l.room}_${l.teacher}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });

    if (uniqueLessons.length === 0) {
      container.innerHTML = '<div class="empty-inline-state">Kein Unterricht verzeichnet.</div>';
      return;
    }

    container.innerHTML = uniqueLessons.map(l => {
      const color = l.color || '#ff7a00';
      const isSubst = l.isSubstitution || l.isRoomChange;
      const isCanc = l.isCancelled;
      const displaySubj = getDisplaySubject(l);
      const roomText = l.room || '';
      const teacherText = l.teacher || '';

      const statusBadge = isCanc 
        ? '<span class="status-pill cancelled">Ausfall</span>' 
        : (isSubst ? '<span class="status-pill substitution">Änderung</span>' : '');

      const storeId = registerLesson(l);

      return `
        <div class="lesson-card ${isSubst ? 'substitution' : ''} ${isCanc ? 'cancelled' : ''}" onclick="openLessonById('${storeId}')">
          <div class="lesson-left-group">
            <div class="lesson-card-badge" style="background-color:${color}; color:${l.textColor || '#ffffff'};">
              ${l.periodNum ? l.periodNum + '.' : (l.subject || '').slice(0, 3)}
            </div>
            <div class="lesson-info-group">
              <span class="lesson-card-title">${escapeHTML(displaySubj)}</span>
              <span class="lesson-card-meta">
                ${teacherText ? `Lehrkraft: <strong>${escapeHTML(teacherText)}</strong>` : ''} 
                ${roomText ? `· Raum: <strong>${escapeHTML(roomText)}</strong>` : ''}
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

  // Default Period Definitions (Vocational & Regular School Standard)
  const DEFAULT_PERIODS = [
    { num: 1, start: '07:30', end: '08:15', label: '1. Std.', range: '07:30 - 08:15' },
    { num: 2, start: '08:15', end: '09:00', label: '2. Std.', range: '08:15 - 09:00' },
    { num: 3, start: '09:15', end: '10:00', label: '3. Std.', range: '09:15 - 10:00' },
    { num: 4, start: '10:00', end: '10:45', label: '4. Std.', range: '10:00 - 10:45' },
    { num: 5, start: '11:00', end: '11:45', label: '5. Std.', range: '11:00 - 11:45' },
    { num: 6, start: '11:45', end: '12:30', label: '6. Std.', range: '11:45 - 12:30' },
    { num: 7, start: '12:45', end: '13:30', label: '7. Std.', range: '12:45 - 13:30' },
    { num: 8, start: '13:30', end: '14:15', label: '8. Std.', range: '13:30 - 14:15' },
    { num: 9, start: '14:30', end: '15:15', label: '9. Std.', range: '14:30 - 15:15' },
    { num: 10, start: '15:15', end: '16:00', label: '10. Std.', range: '15:15 - 16:00' },
    { num: 11, start: '16:15', end: '17:00', label: '11. Std.', range: '16:15 - 17:00' },
    { num: 12, start: '17:00', end: '17:45', label: '12. Std.', range: '17:00 - 17:45' }
  ];

  function getLessonPeriodNumbers(l) {
    let pStart = null;
    let pEnd = null;

    if (l.period) {
      const rangeMatch = l.period.match(/(\d+)\s*\.?\s*-\s*(\d+)\.?/);
      if (rangeMatch) {
        pStart = parseInt(rangeMatch[1], 10);
        pEnd = parseInt(rangeMatch[2], 10);
      } else {
        const singleMatch = l.period.match(/(\d+)\.?/);
        if (singleMatch) {
          pStart = parseInt(singleMatch[1], 10);
          pEnd = pStart;
        }
      }
    }

    if (!pStart && l.periodNum) {
      pStart = l.periodNum;
      pEnd = l.periodNum;
    }

    if (!pStart && l.startTimeStr) {
      const [sh, sm] = l.startTimeStr.split(':').map(Number);
      const sMin = sh * 60 + sm;
      for (const p of DEFAULT_PERIODS) {
        const [psh, psm] = p.start.split(':').map(Number);
        if (Math.abs(sMin - (psh * 60 + psm)) <= 25) {
          pStart = p.num;
          break;
        }
      }
    }

    if (pStart && !pEnd && l.endTimeStr) {
      const [eh, em] = l.endTimeStr.split(':').map(Number);
      const eMin = eh * 60 + em;
      for (const p of DEFAULT_PERIODS) {
        const [peh, pem] = p.end.split(':').map(Number);
        if (Math.abs(eMin - (peh * 60 + pem)) <= 25) {
          pEnd = p.num;
          break;
        }
      }
    }

    if (!pStart) pStart = 1;
    if (!pEnd || pEnd < pStart) pEnd = pStart;

    const res = [];
    for (let i = pStart; i <= pEnd; i++) {
      res.push(i);
    }
    return res;
  }

  function renderWeekTimetable(lessons, headerId, bodyId) {
    const headerEl = document.getElementById(headerId);
    const bodyEl = document.getElementById(bodyId);
    if (!headerEl || !bodyEl) return;

    // Deduplicate lessons
    const seen = new Set();
    const uniqueLessons = (lessons || []).filter(l => {
      const key = `${l.date || l.Date}_${l.startTimeStr}_${l.endTimeStr}_${l.subject}_${l.room}_${l.teacher}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });

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

    // 1. Header: 6 columns: "Std. / Zeit" + 5 Days
    headerEl.innerHTML = `
      <div class="week-time-header-cell">Std. / Zeit</div>
      ${weekDays.map((d, idx) => {
        const iso = formatDateISO(d);
        const isToday = iso === todayStr;
        return `
          <div class="week-day-header-cell ${isToday ? 'today' : ''}">
            <div class="w-day-name">${dayNames[idx]}</div>
            <div class="w-day-date">${d.getDate()}.${d.getMonth() + 1}.</div>
          </div>
        `;
      }).join('')}
    `;

    // If completely empty week
    if (uniqueLessons.length === 0) {
      bodyEl.innerHTML = `
        <div style="grid-column: 1 / -1; padding: 60px 20px; text-align: center; color: var(--text-muted); background: var(--bg-card); border: 1px dashed var(--border-subtle); border-radius: var(--radius-sm);">
          <svg viewBox="0 0 24 24" fill="currentColor" style="width: 44px; height: 44px; opacity: 0.4; margin-bottom: 10px;"><path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2zm-8 4H7v2h2v-2zm4 4H7v2h2v-2zm0-4h-2v2h2v-2zm4 0h-2v2h2v-2z"/></svg>
          <div style="font-size: 15px; font-weight: 700; color: var(--text-primary);">Keine Unterrichtsstunden in dieser Woche</div>
          <div style="font-size: 13px; color: var(--text-secondary); margin-top: 4px;">Für den ausgewählten Zeitraum liegt kein Stundenplan vor (z. B. Ferien oder schulfrei).</div>
        </div>
      `;
      return;
    }

    // Determine max period present (at least 8, max 12)
    let maxPeriod = 8;
    uniqueLessons.forEach(l => {
      const pList = getLessonPeriodNumbers(l);
      pList.forEach(p => {
        if (p > maxPeriod && p <= 12) maxPeriod = p;
      });
    });

    // Map lessons by date and period
    const slotMap = new Map();
    uniqueLessons.forEach(l => {
      const d = l.date || l.Date;
      const periods = getLessonPeriodNumbers(l);
      periods.forEach(p => {
        const slotKey = `${d}_${p}`;
        if (!slotMap.has(slotKey)) {
          slotMap.set(slotKey, []);
        }
        slotMap.get(slotKey).push(l);
      });
    });

    // Render 6-column matrix row-by-row
    const rowsHtml = [];
    for (let p = 1; p <= maxPeriod; p++) {
      const pDef = DEFAULT_PERIODS.find(item => item.num === p) || {
        num: p,
        range: ''
      };

      // Col 1: Time column cell
      rowsHtml.push(`
        <div class="week-time-slot-cell">
          <div class="w-time-num">${p}. Stunde</div>
          <div class="w-time-range">${pDef.range}</div>
        </div>
      `);

      // Col 2..6: 5 Days
      for (let dayIdx = 0; dayIdx < 5; dayIdx++) {
        const iso = formatDateISO(weekDays[dayIdx]);
        const slotKey = `${iso}_${p}`;
        const slotLessons = slotMap.get(slotKey) || [];

        if (slotLessons.length === 0) {
          rowsHtml.push(`
            <div class="week-empty-slot">
              <span>—</span>
            </div>
          `);
        } else {
          const cardsHtml = slotLessons.map(l => {
            const isSubst = l.isSubstitution || l.isRoomChange;
            const isCanc = l.isCancelled;
            const color = l.color || '#ff7a00';
            const displaySubj = getDisplaySubject(l);
            const roomText = l.room || '';
            const teacherText = l.teacher || '';
            const storeId = registerLesson(l);

            return `
              <div class="week-lesson-box ${isSubst ? 'substitution' : ''} ${isCanc ? 'cancelled' : ''}" 
                   style="border-left: 4px solid ${color};"
                   onclick="openLessonById('${storeId}')"
                   title="${escapeHTML(l.subjectLong || displaySubj)}">
                <div class="w-time-row">
                  <span class="w-time">${l.startTimeStr} - ${l.endTimeStr}</span>
                  <span class="w-period">${p}. Std.</span>
                </div>
                <div class="w-subj">${escapeHTML(displaySubj)}</div>
                <div class="w-details">
                  ${roomText ? `<strong>${escapeHTML(roomText)}</strong>` : ''} 
                  ${teacherText ? `· ${escapeHTML(teacherText)}` : ''}
                  ${isSubst ? '<span class="status-subst-badge">(Änderung)</span>' : ''}
                  ${isCanc ? '<span class="status-canc-badge">(Ausfall)</span>' : ''}
                </div>
              </div>
            `;
          }).join('');

          if (slotLessons.length > 1) {
            rowsHtml.push(`<div style="display:flex; flex-direction:column; gap:6px;">${cardsHtml}</div>`);
          } else {
            rowsHtml.push(cardsHtml);
          }
        }
      }
    }

    bodyEl.innerHTML = rowsHtml.join('');
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
              <span class="msg-sender-chip">${escapeHTML(m.senderName || 'Schule')}</span>
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
      <div class="school-result-item" data-school='${escapeHTML(JSON.stringify(s))}' onclick="selectSearchSchool(this)">
        <div>
          <div class="sr-name">${escapeHTML(s.displayName)}</div>
          <div class="sr-details">${escapeHTML(s.address || s.serverUrl || s.server)}</div>
        </div>
        <button class="btn-text-sm">Auswählen &rarr;</button>
      </div>
    `).join('');
  }

  function selectSearchSchool(element) {
    const schoolData = element.getAttribute('data-school');
    if (!schoolData) return;

    try {
      const school = JSON.parse(schoolData);
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
    } catch (err) {}
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

    let serverHost = selectedSearchSchool.server || selectedSearchSchool.serverUrl || '';
    if (serverHost.startsWith('http://') || serverHost.startsWith('https://')) {
      try {
        const u = new URL(serverHost);
        serverHost = u.origin;
      } catch (err) {}
    } else if (serverHost) {
      serverHost = 'https://' + serverHost.split('/')[0];
    }

    showLoading(true, 'Verbinde mit WebUntis...');
    const res = await apiFetch('/api/profiles', {
      method: 'POST',
      body: JSON.stringify({
        school: selectedSearchSchool.loginName || selectedSearchSchool.displayName,
        server: serverHost,
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

  async function saveSchoolAnonymous() {
    if (!selectedSearchSchool) return;

    let serverHost = selectedSearchSchool.server || selectedSearchSchool.serverUrl || '';
    if (serverHost.startsWith('http://') || serverHost.startsWith('https://')) {
      try {
        const u = new URL(serverHost);
        serverHost = u.origin;
      } catch (err) {}
    } else if (serverHost) {
      serverHost = 'https://' + serverHost.split('/')[0];
    }

    const profileName = document.getElementById('newProfileName')?.value.trim() || (selectedSearchSchool.displayName + ' (Anonym)');

    showLoading(true, 'Verbinde anonym mit WebUntis...');
    const res = await apiFetch('/api/profiles', {
      method: 'POST',
      body: JSON.stringify({
        school: selectedSearchSchool.loginName || selectedSearchSchool.displayName,
        server: serverHost,
        name: profileName,
        username: '',
        password: '',
        setActive: true,
      }),
    });
    showLoading(false);

    if (res && res.success) {
      showToast('Anonym als Gast verbunden!');
      window.location.reload();
    } else {
      showToast(res?.message || 'Verbindung fehlgeschlagen', 'error');
    }
  }

  // ==================== ONBOARDING SETUP WIZARD ====================
  let onboardSelectedSchool = null;

  function openOnboardingWizard() {
    const backdrop = document.getElementById('onboardingWizardBackdrop');
    if (!backdrop) return;
    backdrop.style.display = 'flex';
    goToOnboardStep(1);

    const searchInput = document.getElementById('onboardSchoolSearchInput');
    const searchBtn = document.getElementById('btnOnboardSearch');

    if (searchInput && !searchInput.dataset.hasListener) {
      searchInput.dataset.hasListener = 'true';
      searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') handleOnboardSchoolSearch();
      });
    }
    if (searchBtn && !searchBtn.dataset.hasListener) {
      searchBtn.dataset.hasListener = 'true';
      searchBtn.addEventListener('click', handleOnboardSchoolSearch);
    }
  }

  function goToOnboardStep(step) {
    for (let i = 1; i <= 3; i++) {
      const dot = document.getElementById(`onboardStepDot${i}`);
      const content = document.getElementById(`onboardStep${i}Content`);
      if (dot) {
        dot.classList.remove('active', 'completed');
        if (i < step) dot.classList.add('completed');
        if (i === step) dot.classList.add('active');
      }
      if (content) {
        content.style.display = (i === step) ? 'block' : 'none';
      }
    }
    const line1 = document.getElementById('onboardStepLine1');
    const line2 = document.getElementById('onboardStepLine2');
    if (line1) line1.classList.toggle('active', step >= 2);
    if (line2) line2.classList.toggle('active', step >= 3);
  }

  async function handleOnboardSchoolSearch() {
    const input = document.getElementById('onboardSchoolSearchInput');
    const query = input ? input.value.trim() : '';
    if (!query) return;

    const list = document.getElementById('onboardResultsList');
    if (list) list.innerHTML = '<div class="onboard-empty-tip">Suche nach Schulen...</div>';

    const data = await apiFetch(`/api/schools/search?q=${encodeURIComponent(query)}`);
    if (!list) return;

    if (!data || !data.schools || data.schools.length === 0) {
      list.innerHTML = '<div class="onboard-empty-tip">Keine Schule gefunden. Bitte Suchbegriff anpassen.</div>';
      return;
    }

    list.innerHTML = data.schools.map(s => `
      <div class="onboard-school-item" data-school='${escapeHTML(JSON.stringify(s))}' onclick="selectOnboardSchool(this)">
        <div>
          <div class="onboard-school-name">${escapeHTML(s.displayName)}</div>
          <div class="onboard-school-meta">${escapeHTML(s.address || s.serverUrl || s.server)}</div>
        </div>
        <button type="button" class="btn-text-sm">Auswählen &rarr;</button>
      </div>
    `).join('');
  }

  function selectOnboardSchool(element) {
    const schoolData = element.getAttribute('data-school');
    if (!schoolData) return;
    try {
      onboardSelectedSchool = JSON.parse(schoolData);
    } catch {
      return;
    }

    const nameEl = document.getElementById('onboardChosenSchoolName');
    const serverEl = document.getElementById('onboardChosenSchoolServer');
    if (nameEl) nameEl.textContent = onboardSelectedSchool.displayName;
    if (serverEl) serverEl.textContent = onboardSelectedSchool.server || onboardSelectedSchool.serverUrl || 'webuntis.com';

    goToOnboardStep(2);
  }

  async function handleOnboardCredentialsSubmit(e) {
    if (e) e.preventDefault();
    if (!onboardSelectedSchool) return;

    const username = document.getElementById('onboardUsername').value.trim();
    const password = document.getElementById('onboardPassword').value;
    const profileName = document.getElementById('onboardProfileName').value.trim();

    if (!username) {
      showToast('Bitte Benutzernamen eingeben', 'error');
      return;
    }

    let serverHost = onboardSelectedSchool.server || onboardSelectedSchool.serverUrl || '';
    if (serverHost.startsWith('http://') || serverHost.startsWith('https://')) {
      try {
        const u = new URL(serverHost);
        serverHost = u.origin;
      } catch (err) {}
    } else if (serverHost) {
      serverHost = 'https://' + serverHost.split('/')[0];
    }

    showLoading(true, 'Prüfe Zugangsdaten...');
    const res = await apiFetch('/api/profiles', {
      method: 'POST',
      body: JSON.stringify({
        school: onboardSelectedSchool.loginName || onboardSelectedSchool.displayName,
        server: serverHost,
        name: profileName || onboardSelectedSchool.displayName,
        username,
        password,
        setActive: true,
      }),
    });
    showLoading(false);

    if (res && res.success) {
      const subEl = document.getElementById('onboardSuccessSubtitle');
      if (subEl) subEl.textContent = `Angemeldet als ${res.displayName || username}.`;
      goToOnboardStep(3);
    } else {
      showToast(res?.message || 'Anmeldung fehlgeschlagen. Zugangsdaten prüfen.', 'error');
    }
  }

  async function handleOnboardAnonymousSubmit() {
    if (!onboardSelectedSchool) return;

    let serverHost = onboardSelectedSchool.server || onboardSelectedSchool.serverUrl || '';
    if (serverHost.startsWith('http://') || serverHost.startsWith('https://')) {
      try {
        const u = new URL(serverHost);
        serverHost = u.origin;
      } catch (err) {}
    } else if (serverHost) {
      serverHost = 'https://' + serverHost.split('/')[0];
    }

    showLoading(true, 'Verbinde anonym mit WebUntis...');
    const res = await apiFetch('/api/profiles', {
      method: 'POST',
      body: JSON.stringify({
        school: onboardSelectedSchool.loginName || onboardSelectedSchool.displayName,
        server: serverHost,
        name: onboardSelectedSchool.displayName + ' (Anonym)',
        username: '',
        password: '',
        setActive: true,
      }),
    });
    showLoading(false);

    if (res && res.success) {
      const subEl = document.getElementById('onboardSuccessSubtitle');
      if (subEl) subEl.textContent = `Verbunden mit ${onboardSelectedSchool.displayName} als Gast.`;
      goToOnboardStep(3);
    } else {
      showToast(res?.message || 'Verbindung fehlgeschlagen', 'error');
    }
  }

  function finishOnboarding() {
    const backdrop = document.getElementById('onboardingWizardBackdrop');
    if (backdrop) backdrop.style.display = 'none';
    window.location.reload();
  }

  function openProfilesModal() {
    const modal = document.getElementById('profilesModalBackdrop');
    if (modal) {
      modal.style.display = 'flex';
      switchSettingsModalTab('profiles');
      loadProfiles();
      loadSubjectAliases();
    }
  }

  function closeProfilesModal() {
    const modal = document.getElementById('profilesModalBackdrop');
    if (modal) modal.style.display = 'none';
    cancelAddSchool();
  }

  window.switchSettingsModalTab = function(tab) {
    const profTabBtn = document.getElementById('profModalTabProfiles');
    const aliasTabBtn = document.getElementById('profModalTabAliases');
    const profContent = document.getElementById('settingsProfilesTabContent');
    const aliasContent = document.getElementById('settingsAliasesTabContent');

    if (tab === 'aliases') {
      profTabBtn?.classList.remove('active');
      aliasTabBtn?.classList.add('active');
      if (profContent) profContent.style.display = 'none';
      if (aliasContent) aliasContent.style.display = 'block';
      loadSubjectAliases();
    } else {
      aliasTabBtn?.classList.remove('active');
      profTabBtn?.classList.add('active');
      if (aliasContent) aliasContent.style.display = 'none';
      if (profContent) profContent.style.display = 'block';
      loadProfiles();
    }
  };

  // ==================== SUBJECT ALIASES (EIGENE FACHNAMEN) ====================
  async function loadSubjectAliases() {
    const res = await apiFetch('/api/settings/aliases');
    if (res && typeof res === 'object') {
      state.subjectAliases = res;
      renderSubjectAliases();
    }
  }

  function renderSubjectAliases() {
    const container = document.getElementById('aliasesListContainer');
    if (!container) return;

    const keys = Object.keys(state.subjectAliases || {});
    if (keys.length === 0) {
      container.innerHTML = '<div class="empty-inline-state">Noch keine eigenen Fachnamen definiert.</div>';
      return;
    }

    container.innerHTML = keys.map(k => `
      <div class="alias-item-row">
        <div class="alias-map-text">
          <span class="alias-orig-badge">${escapeHTML(k)}</span>
          <span class="alias-arrow">&rarr;</span>
          <span class="alias-custom-val">${escapeHTML(state.subjectAliases[k])}</span>
        </div>
        <button type="button" class="btn-delete-alias" onclick="deleteSubjectAlias('${escapeHTML(k)}')" title="Alias löschen">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
          </svg>
        </button>
      </div>
    `).join('');
  }

  window.handleSaveAlias = async function(e) {
    if (e) e.preventDefault();
    const origInput = document.getElementById('aliasOriginalInput');
    const customInput = document.getElementById('aliasCustomInput');
    const orig = (origInput?.value || '').trim();
    const custom = (customInput?.value || '').trim();

    if (!orig || !custom) return;

    const blockedTerms = ['hitler', 'nazi', 'hakenkreuz', 'swastika', 'vergasen', 'gasdusche', 'gasunterricht', 'gaskammer', 'auschwitz', 'holocaust', 'zyklon', 'arier', 'aryan', 'goebbels', 'himmler', 'eichmann', 'neger', 'kanake', 'judensau', 'siegheil', 'heilhitler'];
    const lower = custom.toLowerCase();
    const norm = lower.replace(/[^a-z0-9]/g, '').replace(/[1!]/g, 'i').replace(/0/g, 'o').replace(/3/g, 'e').replace(/[4@]/g, 'a').replace(/[$5]/g, 's').replace(/7/g, 't');
    for (const b of blockedTerms) {
      if (lower.includes(b) || norm.includes(b)) {
        showToast('Dieser Name enthält unzulässige oder anstößige Begriffe. Bitte wähle eine respektvolle Bezeichnung.', 'error');
        return;
      }
    }

    showLoading(true, 'Speichere Fachnamen...');
    const res = await apiFetch('/api/settings/aliases', {
      method: 'POST',
      body: JSON.stringify({ original: orig, alias: custom }),
    });
    showLoading(false);

    if (res && res.success) {
      showToast(`Fach '${orig}' wird nun als '${custom}' angezeigt.`);
      if (origInput) origInput.value = '';
      if (customInput) customInput.value = '';
      await loadSubjectAliases();
      if (state.currentView === 'dashboard') loadDashboard();
      else if (state.currentView === 'own-timetable') loadOwnTimetable();
      else if (state.currentView === 'other-timetables') loadResourceTimetable();
    } else {
      showToast(res?.message || 'Fehler beim Speichern des Alias', 'error');
    }
  };

  window.deleteSubjectAlias = async function(orig) {
    showLoading(true, 'Lösche Fach-Alias...');
    const res = await apiFetch(`/api/settings/aliases?original=${encodeURIComponent(orig)}`, {
      method: 'DELETE',
    });
    showLoading(false);

    if (res && res.success) {
      showToast(`Alias für '${orig}' entfernt.`);
      await loadSubjectAliases();
      if (state.currentView === 'dashboard') loadDashboard();
      else if (state.currentView === 'own-timetable') loadOwnTimetable();
      else if (state.currentView === 'other-timetables') loadResourceTimetable();
    } else {
      showToast(res?.message || 'Fehler beim Löschen', 'error');
    }
  };

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

    if (currVerEl) currVerEl.textContent = availableUpdateInfo.currentVersion || 'v1.3.1';
    if (newVerEl) newVerEl.textContent = availableUpdateInfo.latestVersion || 'v1.3.1';
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

  // ==================== ÜBER & INFO MODULE ====================
  const APP_RELEASES = [
    {
      version: 'v1.4',
      title: 'Offizieller Release v1.4',
      type: 'release',
      date: '04.09.2026',
      badge: 'Offizieller Release',
      description: 'Erster offizieller Haupt-Release von untis-go! Einführung der interaktiven Info- & Release-Zentrale, Einbindung des offiziellen Marken-Icons, Wiederherstellung des Einzelstunden-Zeitrasters und Aufnahme in Awesome Go.',
      sections: [
        {
          title: '🚀 Features & Neuerungen',
          items: [
            { type: 'feat', text: '<strong>Dedizierte Info- & Release-Seite:</strong> Neuer Navigationsbereich "Über & Info" mit detaillierten Versionsangaben, exakter GitHub-Release-Historie, One-Click Update-Prüfung & Sofortinstallation, Nennung der Mitwirkenden (Jeremy Benz und KI-Pair-Programming-Assistenten Claude Code & Google Antigravity) sowie GNU General Public License v3 Lizenzhinweis.' },
            { type: 'feat', text: '<strong>Aufnahme in Awesome Go (#6660):</strong> untis-go wurde offiziell in das renommierte Verzeichnis <a href="https://github.com/avelino/awesome-go#other-software" target="_blank" rel="noopener">Awesome Go</a> unter <em>Other Software</em> aufgenommen.' },
            { type: 'feat', text: '<strong>Offizielles Untis Marken-Icon:</strong> Einbindung des originalen transparenten Untis-Logos aus dem offiziellen Media-Kit für das native GTK-Desktop-Fenster, Web-App-Favicons (ICO & PNG 512x512) und App-Header.' },
            { type: 'feat', text: '<strong>Synchrone Stundenplan-Matrix & Einzelstunden-Raster:</strong> Wiederherstellung der linken Zeit- & Stundenleiste ("Std. / Zeit"). Mehrstündige Blockstunden (z.B. 4 Stunden am Stück) werden nun sauber und transparent auf jede einzelne Schulstunde aufgeteilt (1. Std., 2. Std., 3. Std., 4. Std.) statt unübersichtlicher Mischbezeichnungen wie "1/2" oder "2/3".' },
          ]
        },
        {
          title: '🐛 Bugfixes & Optimierungen',
          items: [
            { type: 'fix', text: '<strong>Perfekte Zeilenausrichtung im Wochenplan:</strong> Durch die tabellarische CSS-Grid-Matrix sind alle Tage von Montag bis Freitag zeilengenau synchron zu den Schulstunden der Zeitleiste ausgerichtet.' },
            { type: 'fix', text: '<strong>Sicherheits- & Datenschutz-Audit:</strong> Strengere Validierung des 32-Zeichen-Session-Tokens auf allen API-Routen, XSS-Prävention durch strukturierte Objekt-Registrierung und vollständige Anonymisierung sämtlicher Crypto-Salts.' },
          ]
        }
      ]
    },
    {
      version: 'v1.3.1',
      title: 'Hotfix Release v1.3.1',
      type: 'hotfix',
      date: '04.09.2026',
      badge: 'Hotfix',
      description: 'Dringendes Hotfix-Update zur Behebung von Login-Fehlern durch URL-Verunreinigungen und Bereinigung der Crypto-Salts.',
      sections: [
        {
          title: '🐛 Bugfixes',
          items: [
            { type: 'fix', text: '<strong>WebUntis Server-URL Normalisierung:</strong> URLs aus der WebUntis-Schulsuche (mit Pfaden wie /WebUntis/?school=...) werden automatisch zur reinen Basis-Origin-Domain bereinigt. Behebt "NO_MANDANT" Login-Verweigerung.' },
            { type: 'fix', text: '<strong>Aussagekräftige Login-Fehlermeldungen:</strong> Exakte Unterscheidung zwischen ungültigem Schul-Mandant und falschem Passwort.' },
            { type: 'fix', text: '<strong>Vollständige Neutralisierung:</strong> Alle verbliebenen Personenbezüge in Crypto-Salts wurden vollständig entfernt.' },
            { type: 'fix', text: '<strong>Cache-Busting für Go-Proxy:</strong> Version v1.3.1 stellt sicher, dass go install den fehlerbereinigten Stand lädt.' },
          ]
        }
      ]
    },
    {
      version: 'v1.3',
      title: 'Release v1.3 (BETA)',
      type: 'beta',
      date: '04.09.2026',
      badge: 'BETA',
      description: 'Zweites großes Feature-Update mit Profanity-Filter für Aliase, sekundengenauem Countdown-Timer und optimierter Stundenberechnung.',
      sections: [
        {
          title: '🚀 Features',
          items: [
            { type: 'feat', text: '<strong>Filter für respektvolle Fachbezeichnungen:</strong> Automatischer Schutzfilter bei benutzerdefinierten Fach-Aliassen (Profanity & Hate-Speech Filter).' },
            { type: 'feat', text: '<strong>Live-Countdown-Timer mit Sekundenanzeige:</strong> Der Unterrichts- und Pausen-Countdown aktualisiert sich sekundengenau live.' },
            { type: 'feat', text: '<strong>Benutzerdefinierte Fachanzeige:</strong> Bei hinterlegtem Alias wird das Fach übersichtlich als "Wunschname · Kürzel" angezeigt.' },
          ]
        },
        {
          title: '🐛 Bugfixes',
          items: [
            { type: 'fix', text: '<strong>Wochenplan-Layout & Duplikat-Behebung:</strong> Sauberes 5-Tage-Layout ohne überlappende Zeilenraster.' },
            { type: 'fix', text: '<strong>Exakte Mehrstunden-Berechnung:</strong> Blockunterricht wird präzise als vollständige Stundenspanne erkannt (z.B. 3. - 6. Stunde).' },
            { type: 'fix', text: '<strong>Doppelte Fachtitel in Tageskarten behoben:</strong> Anzeige der Stundennummer im Badge verhindert redundante Fachkürzel.' },
          ]
        }
      ]
    },
    {
      version: 'v1.2',
      title: 'Release v1.2 (BETA)',
      type: 'beta',
      date: '04.09.2026',
      badge: 'BETA',
      description: 'Einführung von benutzerdefinierten Fachnamen, übersichtlichen Schulkürzeln und Vertretungs-Hervorhebungen.',
      sections: [
        {
          title: '🚀 Features',
          items: [
            { type: 'feat', text: '<strong>Benutzerdefinierte Fachnamen & Kürzel:</strong> Individuelle Bezeichnungen für Fächer dauerhaft in SQLite sichern.' },
            { type: 'feat', text: '<strong>Standardmäßig Schulkürzel:</strong> Kompakte Originalkürzel der Schule (FB02, E, LF01, FP, WBL).' },
            { type: 'feat', text: '<strong>Reine Raumnummern:</strong> Glatte Raumnummern wie F108, H119, J103 ohne Textballast.' },
            { type: 'feat', text: '<strong>Vertretungs- & Ausfall-Hervorhebung:</strong> Vertretungen mit Akzent, Ausfälle durchgestrichen.' },
          ]
        },
        {
          title: '🐛 Bugfixes',
          items: [
            { type: 'fix', text: '<strong>Deduplizierung im Stundenplan:</strong> Filterung von Mehrfacheinträgen bei Koppelkursen.' },
            { type: 'fix', text: '<strong>Dashboard-Metriken bereinigt:</strong> Redundante Metrik-Karten zusammengeführt.' },
          ]
        }
      ]
    },
    {
      version: 'v1.1',
      title: 'Release v1.1 (BETA)',
      type: 'beta',
      date: '04.09.2026',
      badge: 'BETA',
      description: 'Integrierte Auto-Update Engine, Raum-Stundenplan-Berechnung, Klassen-Favoriten und Gelesen-Status für Mitteilungen.',
      sections: [
        {
          title: '🚀 Features',
          items: [
            { type: 'feat', text: '<strong>In-App Auto-Update-System:</strong> Hintergrundprüfung auf GitHub-Releases mit One-Click-Aktualisierung.' },
            { type: 'feat', text: '<strong>Raum-Stundenpläne berechnen:</strong> Freie und belegte Räume intelligent aus Stundenplandaten ermitteln.' },
            { type: 'feat', text: '<strong>Klassen-Favoriten (❤️):</strong> Klassen in "Weitere Stundenpläne" oben anpinnen.' },
            { type: 'feat', text: '<strong>Gelesen-Status bei Mitteilungen:</strong> Lokale Speicherung und Button "Alle als gelesen markieren".' },
          ]
        },
        {
          title: '🐛 Bugfixes',
          items: [
            { type: 'fix', text: '<strong>Parser-Korrektur:</strong> Dynamische Zuordnung von Fach, Lehrer, Raum und Klasse repariert.' },
            { type: 'fix', text: '<strong>Wochenansicht-Korrektur:</strong> Problem mit fälschlichem "Frei"-Zustand gelöst.' },
          ]
        }
      ]
    },
    {
      version: 'v1.0',
      title: 'Release v1.0 (BETA)',
      type: 'beta',
      date: '03.09.2026',
      badge: 'BETA',
      description: 'Erster öffentlicher BETA-Release von untis-go als blitzschneller nativer WebUntis Desktop-Client.',
      sections: [
        {
          title: '✨ Kernfunktionen & Module',
          items: [
            { type: 'feat', text: '<strong>Dashboard:</strong> Tagesbegrüßung, Stundenplan-Übersicht, offene Hausaufgaben & ungelesene Mitteilungen.' },
            { type: 'feat', text: '<strong>Mein Stundenplan:</strong> Schnelle Tages- und Wochenansicht mit Live-Marker.' },
            { type: 'feat', text: '<strong>Hausaufgaben & Abwesenheiten:</strong> Aufgaben abhaken und Krankmeldungen eintragen.' },
            { type: 'feat', text: '<strong>Sicherheit & SQLite Cache:</strong> AES-256-GCM Verschlüsselung, dynamischer Port und Offline-Cache.' },
          ]
        }
      ]
    }
  ];

  function loadAboutView() {
    renderAboutReleases();
  }

  function renderAboutReleases() {
    const listEl = document.getElementById('aboutReleasesList');
    if (!listEl) return;

    listEl.innerHTML = APP_RELEASES.map((rel, idx) => {
      const isLatest = idx === 0;
      const pillClass = rel.type === 'release' ? 'release' : (rel.type === 'hotfix' ? 'hotfix' : 'beta');

      const sectionsHtml = (rel.sections || []).map(sec => `
        <div class="about-rel-section-title">${sec.title}</div>
        <ul class="about-rel-list">
          ${sec.items.map(item => `
            <li>
              <span class="about-tag-${item.type}">${item.type === 'feat' ? '[FEATURE]' : '[BUGFIX]'}</span>
              <span>${item.text}</span>
            </li>
          `).join('')}
        </ul>
      `).join('');

      return `
        <div class="about-release-item ${isLatest ? 'latest' : ''}">
          <div class="about-release-header">
            <div class="about-release-tag-group">
              <span class="about-rel-version">${escapeHTML(rel.version)}</span>
              <span class="about-rel-pill ${pillClass}">${escapeHTML(rel.badge)}</span>
            </div>
            <span class="about-rel-date">${escapeHTML(rel.date)}</span>
          </div>
          <div class="about-rel-body">
            <p style="margin: 0 0 10px; font-weight: 500; color: var(--text-primary);">${escapeHTML(rel.description)}</p>
            ${sectionsHtml}
          </div>
        </div>
      `;
    }).join('');
  }

  async function checkForUpdatesFromAbout() {
    const btn = document.getElementById('btnAboutCheckUpdate');
    const installBtn = document.getElementById('btnAboutInstallUpdate');
    const msgEl = document.getElementById('aboutUpdateStatusMsg');

    if (btn) {
      btn.disabled = true;
      btn.classList.add('loading');
    }
    if (msgEl) msgEl.textContent = 'Suche nach Updates auf GitHub...';

    try {
      const res = await apiFetch('/api/updates/check');
      if (res && res.hasUpdate) {
        availableUpdateInfo = res;
        if (installBtn) installBtn.style.display = 'inline-flex';
        if (msgEl) {
          msgEl.innerHTML = `Neue Version verfügbar: <strong style="color: var(--accent-primary);">${escapeHTML(res.latestVersion)}</strong>!`;
        }
        showToast(`Update ${res.latestVersion} verfügbar!`, 'success');
        openUpdateModal();
      } else {
        if (installBtn) installBtn.style.display = 'none';
        if (msgEl) {
          msgEl.innerHTML = `untis-go ist auf dem neuesten Stand (<strong>${escapeHTML(res?.currentVersion || 'v1.4')}</strong>).`;
        }
        showToast(`untis-go ist auf dem neuesten Stand (${res?.currentVersion || 'v1.4'}).`);
      }
    } catch (e) {
      if (msgEl) msgEl.textContent = 'Fehler bei der Update-Prüfung.';
      showToast('Fehler beim Prüfen auf Updates', 'error');
    } finally {
      if (btn) {
        btn.disabled = false;
        btn.classList.remove('loading');
      }
    }
  }

  // ==================== LESSON DETAIL MODAL ====================
  function openLessonDetailModal(lesson) {
    const modal = document.getElementById('lessonDetailBackdrop');
    if (!modal) return;

    const chip = document.getElementById('lessonModalSubjChip');
    const displaySubj = getDisplaySubject(lesson);
    if (chip) {
      chip.textContent = (displaySubj || 'LF').slice(0, 4);
      chip.style.backgroundColor = lesson.color || '#ff7a00';
      chip.style.color = lesson.textColor || '#ffffff';
    }

    const fullSubjectTitle = displaySubj + (lesson.subjectLong && lesson.subjectLong !== displaySubj ? ` (${lesson.subjectLong})` : '');
    document.getElementById('lessonModalSubject').textContent = fullSubjectTitle;
    document.getElementById('lessonModalTime').textContent = `${lesson.timeRange} (${lesson.period})`;
    document.getElementById('lessonModalTeacher').textContent = lesson.teacherLong || lesson.teacher || '-';
    document.getElementById('lessonModalRoom').textContent = lesson.room || '-';
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
    document.getElementById('navAbout')?.addEventListener('click', () => switchView('about'));

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
  window.openLessonById = openLessonById;
  window.loadAboutView = loadAboutView;
  window.checkForUpdatesFromAbout = checkForUpdatesFromAbout;
  window.openOnboardingWizard = openOnboardingWizard;
  window.goToOnboardStep = goToOnboardStep;
  window.handleOnboardSchoolSearch = handleOnboardSchoolSearch;
  window.selectOnboardSchool = selectOnboardSchool;
  window.handleOnboardCredentialsSubmit = handleOnboardCredentialsSubmit;
  window.handleOnboardAnonymousSubmit = handleOnboardAnonymousSubmit;
  window.finishOnboarding = finishOnboarding;
  window.saveSchoolAnonymous = saveSchoolAnonymous;

  // Run app on DOMContentLoaded
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initApp);
  } else {
    initApp();
  }
})();
