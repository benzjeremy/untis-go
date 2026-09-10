//go:build linux && cgo

package main

/*
#cgo pkg-config: ayatana-appindicator3-0.1 webkit2gtk-4.1 gtk+-3.0
#include <stdlib.h>
#include <gtk/gtk.h>
#include <webkit2/webkit2.h>
#include <libayatana-appindicator/app-indicator.h>

static int check_display() {
    int argc = 0;
    char **argv = NULL;
    return gtk_init_check(&argc, &argv) ? 1 : 0;
}

static void on_window_destroy(GtkWidget *widget, gpointer data) {
    gtk_main_quit();
}

static void on_title_changed(WebKitWebView *web_view, GParamSpec *pspec, GtkWindow *window) {
    const gchar *new_title = webkit_web_view_get_title(web_view);
    if (new_title && *new_title) {
        gtk_window_set_title(window, new_title);
    }
}

static void set_window_icon_from_memory(GtkWindow *window, const void *buf, gsize len) {
    if (!buf || len == 0) return;
    GError *err = NULL;
    GdkPixbufLoader *loader = gdk_pixbuf_loader_new();
    if (loader) {
        if (gdk_pixbuf_loader_write(loader, (const guint8 *)buf, len, &err)) {
            gdk_pixbuf_loader_close(loader, &err);
            GdkPixbuf *pixbuf = gdk_pixbuf_loader_get_pixbuf(loader);
            if (pixbuf) {
                gtk_window_set_icon(window, pixbuf);
                gtk_window_set_default_icon(pixbuf);
            }
        }
        g_object_unref(loader);
    }
    gtk_window_set_default_icon_name("untis-go");
    gtk_window_set_icon_name(window, "untis-go");
}

static gboolean on_context_menu(WebKitWebView *web_view, WebKitContextMenu *context_menu, GdkEvent *event, WebKitHitTestResult *hit_test_result, gpointer user_data) {
    // Native app feeling: suppress browser context menu
    return TRUE;
}

static gboolean on_script_dialog(WebKitWebView *web_view, WebKitScriptDialog *dialog, gpointer user_data) {
    // Suppress WebKit default alert/confirm/prompt browser dialogs
    return TRUE;
}

static gboolean on_window_delete(GtkWidget *widget, GdkEvent *event, gpointer user_data) {
    gtk_widget_hide(widget);
    return TRUE; // Minimize to system tray instead of quitting
}

static void on_tray_open(GtkMenuItem *item, gpointer user_data) {
    GtkWidget *window = GTK_WIDGET(user_data);
    gtk_window_present(GTK_WINDOW(window));
}

static void on_tray_reload(GtkMenuItem *item, gpointer user_data) {
    WebKitWebView *webview = WEBKIT_WEB_VIEW(user_data);
    webkit_web_view_reload(webview);
}

static void on_tray_quit(GtkMenuItem *item, gpointer user_data) {
    gtk_main_quit();
}

#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
static void setup_tray_indicator(GtkWidget *window, GtkWidget *webview) {
    AppIndicator *indicator = app_indicator_new("untis-go", "untis-go", APP_INDICATOR_CATEGORY_APPLICATION_STATUS);
    if (!indicator) return;

    app_indicator_set_status(indicator, APP_INDICATOR_STATUS_ACTIVE);
    app_indicator_set_title(indicator, "untis-go");

    GtkWidget *menu = gtk_menu_new();

    GtkWidget *item_open = gtk_menu_item_new_with_label("📅 Stundenplan öffnen");
    g_signal_connect(item_open, "activate", G_CALLBACK(on_tray_open), window);
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), item_open);

    GtkWidget *item_reload = gtk_menu_item_new_with_label("🔄 Stundenplan neu laden");
    g_signal_connect(item_reload, "activate", G_CALLBACK(on_tray_reload), webview);
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), item_reload);

    GtkWidget *sep1 = gtk_separator_menu_item_new();
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), sep1);

    GtkWidget *item_status = gtk_menu_item_new_with_label("🟢 Status: Verbunden");
    gtk_widget_set_sensitive(item_status, FALSE);
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), item_status);

    GtkWidget *sep2 = gtk_separator_menu_item_new();
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), sep2);

    GtkWidget *item_quit = gtk_menu_item_new_with_label("❌ Beenden");
    g_signal_connect(item_quit, "activate", G_CALLBACK(on_tray_quit), NULL);
    gtk_menu_shell_append(GTK_MENU_SHELL(menu), item_quit);

    gtk_widget_show_all(menu);
    app_indicator_set_menu(indicator, GTK_MENU(menu));
}
#pragma GCC diagnostic pop

static void run_gtk_window(const char *title, const char *url, int width, int height, const void *icon_buf, int icon_len) {
    int argc = 0;
    char **argv = NULL;
    if (!gtk_init_check(&argc, &argv)) {
        return;
    }

    g_set_prgname("untis-go");
    g_set_application_name("Untis Desktop");

    GtkWidget *window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    gtk_window_set_title(GTK_WINDOW(window), title);
    gtk_window_set_default_size(GTK_WINDOW(window), width, height);
    gtk_window_set_position(GTK_WINDOW(window), GTK_WIN_POS_CENTER);

    if (icon_buf && icon_len > 0) {
        set_window_icon_from_memory(GTK_WINDOW(window), icon_buf, (gsize)icon_len);
    }

    g_signal_connect(window, "delete-event", G_CALLBACK(on_window_delete), NULL);
    g_signal_connect(window, "destroy", G_CALLBACK(on_window_destroy), NULL);

    GtkWidget *webview = webkit_web_view_new();
    WebKitSettings *settings = webkit_web_view_get_settings(WEBKIT_WEB_VIEW(webview));
    webkit_settings_set_enable_developer_extras(settings, FALSE);
    webkit_settings_set_enable_javascript(settings, TRUE);
    webkit_settings_set_enable_webgl(settings, TRUE);
    webkit_settings_set_enable_2d_canvas_acceleration(settings, TRUE);
    webkit_settings_set_hardware_acceleration_policy(settings, WEBKIT_HARDWARE_ACCELERATION_POLICY_ALWAYS);

    g_signal_connect(webview, "notify::title", G_CALLBACK(on_title_changed), window);
    g_signal_connect(webview, "context-menu", G_CALLBACK(on_context_menu), NULL);
    g_signal_connect(webview, "script-dialog", G_CALLBACK(on_script_dialog), NULL);

    gtk_container_add(GTK_CONTAINER(window), webview);
    webkit_web_view_load_uri(WEBKIT_WEB_VIEW(webview), url);

    setup_tray_indicator(window, webview);

    gtk_widget_show_all(window);
    gtk_main();
}
*/
import "C"
import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"unsafe"

	"github.com/benzjeremy/untis-go/web"
)

func init() {
	_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	_ = os.Setenv("WEBKIT_FORCE_COMPOSITING_MODE", "1")
}

// installDesktopIntegration automatically installs icons and desktop file into user's XDG directories
func installDesktopIntegration() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	iconDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "512x512", "apps")
	pixmapDir := filepath.Join(home, ".local", "share", "pixmaps")
	appDir := filepath.Join(home, ".local", "share", "applications")
	_ = os.MkdirAll(iconDir, 0755)
	_ = os.MkdirAll(pixmapDir, 0755)
	_ = os.MkdirAll(appDir, 0755)

	iconPng, _ := web.Assets.ReadFile("icon.png")
	if len(iconPng) > 0 {
		_ = os.WriteFile(filepath.Join(iconDir, "untis-go.png"), iconPng, 0644)
		_ = os.WriteFile(filepath.Join(pixmapDir, "untis-go.png"), iconPng, 0644)
	}
	iconSvg, _ := web.Assets.ReadFile("icon.svg")
	if len(iconSvg) > 0 {
		svgDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "scalable", "apps")
		_ = os.MkdirAll(svgDir, 0755)
		_ = os.WriteFile(filepath.Join(svgDir, "untis-go.svg"), iconSvg, 0644)
	}

	desktopPath := filepath.Join(appDir, "untis-go.desktop")
	execPath, _ := os.Executable()
	if execPath == "" {
		execPath = "untis-go"
	}
	content := fmt.Sprintf(`[Desktop Entry]
Name=Untis Stundenplan
Comment=Untis Stundenplan Desktop-Anwendung
Exec=%s
Icon=untis-go
Terminal=false
Type=Application
Categories=Education;Office;
StartupWMClass=untis-go
X-Wayland-AppID=untis-go
`, execPath)
	_ = os.WriteFile(desktopPath, []byte(content), 0644)
}

// LaunchGUI attempts to open a native WebKitGTK window, falling back to a browser in app mode
func LaunchGUI(title, url string, width, height int, forceBrowser bool) {
	installDesktopIntegration()
	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""

	if forceBrowser || !hasDisplay {
		if !hasDisplay {
			log.Println("[GUI] Kein Display erkannt (DISPLAY / WAYLAND_DISPLAY leer).")
		}
		OpenBrowser(url)
		return
	}

	if C.check_display() == 0 {
		log.Println("[GUI] Display-Initialisierung fehlgeschlagen. Starte Browser-Modus...")
		OpenBrowser(url)
		return
	}

	cTitle := C.CString(title)
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cURL))

	iconBytes, _ := web.Assets.ReadFile("icon.png")
	var iconPtr unsafe.Pointer
	if len(iconBytes) > 0 {
		iconPtr = unsafe.Pointer(&iconBytes[0])
	}

	log.Println("[GUI] Starte natives WebKitGTK/GTK3-Fenster (60 FPS Hardware-Compositing)...")
	C.run_gtk_window(cTitle, cURL, C.int(width), C.int(height), iconPtr, C.int(len(iconBytes)))
}

// OpenBrowser opens the URL in the system browser in application mode if possible
func OpenBrowser(url string) {
	commands := [][]string{
		{"google-chrome", "--app=" + url},
		{"chromium", "--app=" + url},
		{"brave-browser", "--app=" + url},
		{"firefox", "--new-window", url},
		{"xdg-open", url},
	}

	for _, cmdArgs := range commands {
		if path, err := exec.LookPath(cmdArgs[0]); err == nil {
			cmd := exec.Command(path, cmdArgs[1:]...)
			if err := cmd.Start(); err == nil {
				log.Printf("[Browser] Geöffnet mit %s (%s)\n", cmdArgs[0], url)
				return
			}
		}
	}

	log.Printf("[Browser] Bitte öffne diese URL in deinem Browser: %s\n", url)
}
