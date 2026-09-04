// i18n.js - Simple internationalization for Untis GO
const i18n = {
    languages: ['en', 'de'],
    defaultLang: 'en',
    translations: {
        en: {
            // Navigation
            "nav.dashboard": "Overview",
            "nav.ownTimetable": "My Timetable",
            "nav.otherTimetables": "Other Timetables",
            "nav.homework": "Homework",
            "nav.absences": "Absences",
            "nav.messages": "Messages",
            "nav.profiles": "Profiles",
            "nav.about": "About & Info",
            // Sidebar headers
            "sidebar.schoolPill.title": "Active School",
            "sidebar.refresh.title": "Reload / Synchronize",
            "sidebar.theme.title": "Toggle Color Scheme (Dark / Light)",
            "sidebar.language.title": "Language / Sprache",
            // Topbar
            "topbar.updateBadge.title": "New update available!",
            // Dialog titles
            "modal.homework.title": "+ New Homework",
            "modal.absence.title": "+ Absence / Sick Note",
            "modal.messageDetail.title": "Message Details",
            "modal.lessonDetail.title": "Lesson Details",
            "modal.profiles.title": "Settings & Profiles",
            "modal.update.title": "Software Update",
            // Buttons
            "button.close": "Close",
            "button.cancel": "Cancel",
            "button.save": "Save",
            "button.submit": "Submit",
            "button.add": "+ Add",
            "button.updateNow": "Update Now",
            "button.later": "Later",
            "button.search": "Search",
            // Forms placeholders
            "form.homework.subject": "Subject (e.g. LF09, Mathematics, German)",
            "form.homework.description": "Task / Description",
            "form.homework.dueDate": "Due Date",
            "form.absence.reason": "Reason for Absence",
            "form.absence.text": "Explanation / Remark",
            "form.absence.startDate": "From Date",
            "form.absence.endDate": "To Date",
            "form.absence.excused": "Mark as 'Excused'",
            "form.message.detail.sender": "Sender",
            "form.message.detail.date": "Date",
            "form.message.detail.subject": "Subject",
            "form.message.detail.content": "Content...",
            "form.lesson.detail.subject": "Subject",
            "form.lesson.detail.time": "Time",
            "form.lesson.detail.teacher": "Teacher",
            "form.lesson.detail.room": "Room",
            "form.lesson.detail.class": "Class",
            "form.lesson.detail.status": "Status",
            // Toast messages
            "toast.sessionExpired": "Session expired or invalid token",
            // Misc
            "label.moreLanguagesSoon": "More languages coming soon"
        },
        de: {
            // Navigation
            "nav.dashboard": "Übersicht",
            "nav.ownTimetable": "Mein Stundenplan",
            "nav.otherTimetables": "Weitere Stundenpläne",
            "nav.homework": "Hausaufgaben",
            "nav.absences": "Abwesenheiten",
            "nav.messages": "Mitteilungen",
            "nav.profiles": "Profile",
            "nav.about": "Über & Info",
            // Sidebar headers
            "sidebar.schoolPill.title": "Aktive Schule",
            "sidebar.refresh.title": "Neu laden / Synchronisieren",
            "sidebar.theme.title": "Farbschema umschalten (Dunkel / Hell)",
            "sidebar.language.title": "Language / Sprache",
            // Topbar
            "topbar.updateBadge.title": "Neues Update verfügbar!",
            // Dialog titles
            "modal.homework.title": "+ Neue Hausaufgabe eintragen",
            "modal.absence.title": "+ Abwesenheit / Krankmeldung eintragen",
            "modal.messageDetail.title": "Nachrichtendetails",
            "modal.lessonDetail.title": "Unterrichtsstunde Details",
            "modal.profiles.title": "Einstellungen & Profile",
            "modal.update.title": "Software-Update",
            // Buttons
            "button.close": "Schließen",
            "button.cancel": "Abbrechen",
            "button.save": "Speichern",
            "button.submit": "Absenden",
            "button.add": "+ Hinzufügen",
            "button.updateNow": "Jetzt aktualisieren",
            "button.later": "Später",
            "button.search": "Suchen",
            // Forms placeholders
            "form.homework.subject": "Schulfach (z.B. LF09, Mathematik, Deutsch)",
            "form.homework.description": "Aufgabe / Beschreibung",
            "form.homework.dueDate": "Fälligkeitsdatum",
            "form.absence.reason": "Grund der Abwesenheit",
            "form.absence.text": "Begründung / Bemerkung",
            "form.absence.startDate": "Von Datum",
            "form.absence.endDate": "Bis Datum",
            "form.absence.excused": "Als 'Entschuldigt' markieren",
            "form.message.detail.sender": "Absender",
            "form.message.detail.date": "Datum",
            "form.message.detail.subject": "Betreff",
            "form.message.detail.content": "Inhalt...",
            "form.lesson.detail.subject": "Fach",
            "form.lesson.detail.time": "Zeit",
            "form.lesson.detail.teacher": "Lehrkraft",
            "form.lesson.detail.room": "Raum",
            "form.lesson.detail.class": "Klasse",
            "form.lesson.detail.status": "Status",
            // Toast messages
            "toast.sessionExpired": "Sitzung abgelaufen oder ungültiges Token",
            // Misc
            "label.moreLanguagesSoon": "Weitere Sprachen folgen"
        }
    },
    getCurrentLang: function() {
        const stored = localStorage.getItem('untis_language');
        if (this.languages.includes(stored)) return stored;
        // fallback to browser language
        const lang = navigator.language.split('-')[0];
        if (this.languages.includes(lang)) return lang;
        return this.defaultLang;
    },
    setLang: function(lang) {
        if (this.languages.includes(lang)) {
            localStorage.setItem('untis_language', lang);
            // reload to apply changes
            location.reload();
        }
    },
    t: function(key) {
        const lang = this.getCurrentLang();
        return this.translations[lang][key] || `[${key}]`;
    },
    init: function() {
        const lang = this.getCurrentLang();
        document.documentElement.setAttribute('lang', lang);
        // Translate all elements with data-i18n attribute
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.getAttribute('data-i18n');
            const value = this.t(key);
            // Prefer to set textContent, but if element has placeholder attribute, set that too
            if (el.placeholder !== undefined) {
                el.placeholder = value;
            }
            if (el.title !== undefined && el.hasAttribute('data-i18n-title')) {
                el.title = this.t(el.getAttribute('data-i18n-title'));
            } else {
                el.textContent = value;
            }
        });
    }
};