//go:build linux && cgo

package main

/*
#cgo pkg-config: webkit2gtk-4.1 gtk+-3.0
#include <stdlib.h>
#include <gtk/gtk.h>
#include <webkit2/webkit2.h>

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
            }
        }
        g_object_unref(loader);
    }
}

static gboolean on_context_menu(WebKitWebView *web_view, WebKitContextMenu *context_menu, GdkEvent *event, WebKitHitTestResult *hit_test_result, gpointer user_data) {
    // Native app feeling: suppress browser context menu
    return TRUE;
}

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

    gtk_container_add(GTK_CONTAINER(window), webview);
    webkit_web_view_load_uri(WEBKIT_WEB_VIEW(webview), url);

    gtk_widget_show_all(window);
    gtk_main();
}
*/
import "C"
import (
	"log"
	"os"
	"os/exec"
	"unsafe"

	"github.com/benzjeremy/untis-go/web"
)

func init() {
	_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	_ = os.Setenv("WEBKIT_FORCE_COMPOSITING_MODE", "1")
}

// LaunchGUI attempts to open a native WebKitGTK window, falling back to a browser in app mode
func LaunchGUI(title, url string, width, height int, forceBrowser bool) {
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
