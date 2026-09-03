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

static void run_gtk_window(const char *title, const char *url, int width, int height) {
    int argc = 0;
    char **argv = NULL;
    if (!gtk_init_check(&argc, &argv)) {
        return;
    }

    GtkWidget *window = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    gtk_window_set_title(GTK_WINDOW(window), title);
    gtk_window_set_default_size(GTK_WINDOW(window), width, height);
    gtk_window_set_position(GTK_WINDOW(window), GTK_WIN_POS_CENTER);

    g_signal_connect(window, "destroy", G_CALLBACK(on_window_destroy), NULL);

    GtkWidget *webview = webkit_web_view_new();
    WebKitSettings *settings = webkit_web_view_get_settings(WEBKIT_WEB_VIEW(webview));
    webkit_settings_set_enable_developer_extras(settings, TRUE);
    webkit_settings_set_enable_javascript(settings, TRUE);
    webkit_settings_set_enable_webgl(settings, TRUE);
    webkit_settings_set_enable_2d_canvas_acceleration(settings, TRUE);
    webkit_settings_set_hardware_acceleration_policy(settings, WEBKIT_HARDWARE_ACCELERATION_POLICY_ALWAYS);

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
)

func init() {
	// Disable DMABUF renderer in WebKitGTK to prevent Wayland stutter/flickering
	// and force hardware compositing for smooth 60 FPS
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

	log.Println("[GUI] Starte natives WebKitGTK/GTK3-Fenster (60 FPS Hardware-Compositing)...")
	C.run_gtk_window(cTitle, cURL, C.int(width), C.int(height))
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
