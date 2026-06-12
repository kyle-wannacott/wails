//go:build linux && cgo && !android

package application

/*
#cgo linux pkg-config: gtk4 webkitgtk-6.0

#include <gtk/gtk.h>
#include <webkit/webkit.h>
#include <stdio.h>
#include <stdlib.h>

// Create a new WebKitWebView for a panel
static GtkWidget* panel_new_webview() {
    return webkit_web_view_new();
}

// Create a fixed container to hold the panel webview at specific position
static GtkWidget* panel_new_fixed() {
    return gtk_fixed_new();
}

// Add webview to fixed container at position
static void panel_fixed_put(GtkWidget *fixed, GtkWidget *webview, int x, int y) {
    gtk_fixed_put(GTK_FIXED(fixed), webview, x, y);
}

// Move webview in fixed container
static void panel_fixed_move(GtkWidget *fixed, GtkWidget *webview, int x, int y) {
    gtk_fixed_move(GTK_FIXED(fixed), webview, x, y);
}

// Set webview size
static void panel_set_size(GtkWidget *webview, int width, int height) {
    gtk_widget_set_size_request(webview, width, height);
}

// Load URL in webview
static void panel_load_url(GtkWidget *webview, const char *url) {
    webkit_web_view_load_uri(WEBKIT_WEB_VIEW(webview), url);
}

// Reload webview
static void panel_reload(GtkWidget *webview) {
    webkit_web_view_reload(WEBKIT_WEB_VIEW(webview));
}

// Force reload webview (bypass cache)
static void panel_force_reload(GtkWidget *webview) {
    webkit_web_view_reload_bypass_cache(WEBKIT_WEB_VIEW(webview));
}

// Show webview
static void panel_show(GtkWidget *webview) {
    gtk_widget_set_visible(webview, TRUE);
}

// Hide webview
static void panel_hide(GtkWidget *webview) {
    gtk_widget_set_visible(webview, FALSE);
}

// Check if visible
static gboolean panel_is_visible(GtkWidget *webview) {
    return gtk_widget_get_visible(webview);
}

// Set zoom level
static void panel_set_zoom(GtkWidget *webview, double zoom) {
    webkit_web_view_set_zoom_level(WEBKIT_WEB_VIEW(webview), zoom);
}

// Get zoom level
static double panel_get_zoom(GtkWidget *webview) {
    return webkit_web_view_get_zoom_level(WEBKIT_WEB_VIEW(webview));
}

// Open inspector
static void panel_open_devtools(GtkWidget *webview) {
    WebKitWebInspector *inspector = webkit_web_view_get_inspector(WEBKIT_WEB_VIEW(webview));
    webkit_web_inspector_show(inspector);
}

// Focus webview
static void panel_focus(GtkWidget *webview) {
    gtk_widget_grab_focus(webview);
}

// Check if focused
static gboolean panel_is_focused(GtkWidget *webview) {
    return gtk_widget_has_focus(webview);
}

// Set background color
static void panel_set_background_color(GtkWidget *webview, int r, int g, int b, int a) {
    GdkRGBA color;
    color.red = r / 255.0;
    color.green = g / 255.0;
    color.blue = b / 255.0;
    color.alpha = a / 255.0;
    webkit_web_view_set_background_color(WEBKIT_WEB_VIEW(webview), &color);
}

// Enable/disable devtools
static void panel_enable_devtools(GtkWidget *webview, gboolean enable) {
    WebKitSettings *settings = webkit_web_view_get_settings(WEBKIT_WEB_VIEW(webview));
    webkit_settings_set_enable_developer_extras(settings, enable);
}

// Enable/disable input capture on the panel widget.
// When FALSE, clicks pass through to the main webview below.
static void panel_set_can_target(GtkWidget *widget, int enabled) {
    gtk_widget_set_can_target(widget, enabled);
}

// Destroy the panel webview
// GTK4 removed gtk_widget_destroy; use g_object_unref instead.
static void panel_destroy(GtkWidget *webview) {
    g_object_unref(webview);
}

// Get position allocation
// GTK4: gtk_widget_get_allocation is deprecated; use gtk_widget_compute_bounds.
static void panel_get_allocation(GtkWidget *webview, int *x, int *y, int *width, int *height) {
    graphene_rect_t bounds;
    // Compute bounds relative to the widget's parent
    if (gtk_widget_compute_bounds(webview, gtk_widget_get_parent(webview), &bounds)) {
        *x = (int)bounds.origin.x;
        *y = (int)bounds.origin.y;
        *width = (int)bounds.size.width;
        *height = (int)bounds.size.height;
    }
}

*/
import "C"
import (
	"unsafe"
)

type linuxPanelImpl struct {
	panel   *WebviewPanel
	webview *C.GtkWidget
	fixed   *C.GtkWidget // Fixed container to position the webview
	parent  *linuxWebviewWindow
}

func newPanelImpl(panel *WebviewPanel) webviewPanelImpl {
	parentWindow := panel.parent
	if parentWindow == nil || parentWindow.impl == nil {
		return nil
	}

	linuxParent, ok := parentWindow.impl.(*linuxWebviewWindow)
	if !ok {
		return nil
	}

	return &linuxPanelImpl{
		panel:  panel,
		parent: linuxParent,
	}
}

func (p *linuxPanelImpl) create() {
	options := p.panel.options

	// Create the webview
	p.webview = C.panel_new_webview()
	C.panel_set_size(p.webview, C.int(options.Width), C.int(options.Height))

	// Create a GtkFixed container for absolute positioning
	p.fixed = C.panel_new_fixed()
	C.panel_fixed_put(p.fixed, p.webview, C.int(options.X), C.int(options.Y))

	// Use GtkOverlay so the panel floats above the main webview instead
	// of being packed into the vbox layout (which breaks the app layout).
	//
	// Strategy: remove the main webview from the vbox, create an overlay
	// containing the vbox as the base child, add the panel as overlay child,
	// then put the overlay where the webview was in the vbox.
	vbox := (*C.GtkBox)(p.parent.vbox)

	// Find the main webview widget within the vbox
	mainWebview := p.parent.webview

	// Create overlay and add the vbox as its base child
	overlay := C.gtk_overlay_new()
	C.gtk_widget_set_vexpand(overlay, 1)
	C.gtk_widget_set_hexpand(overlay, 1)
	overlayCast := (*C.GtkOverlay)(unsafe.Pointer(overlay))

	// Add the panel as an overlay child on top
	C.gtk_overlay_add_overlay(overlayCast, p.fixed)

	// Now replace the main webview in the vbox with the overlay.
	// The overlay's base child will be the webview, and the panel
	// will overlay on top at the specified coordinates.
	// We need to remove the webview from the vbox and insert the overlay.

	// GTK4: remove the webview from the vbox
	C.gtk_box_remove(vbox, (*C.GtkWidget)(mainWebview))

	// Add the webview as the base child of the overlay
	C.gtk_overlay_set_child(overlayCast, (*C.GtkWidget)(mainWebview))

	// Insert the overlay into the vbox where the webview was
	C.gtk_box_append(vbox, overlay)

	// Enable devtools if in debug mode
	debugMode := globalApplication.isDebugMode
	devToolsEnabled := debugMode
	if options.DevToolsEnabled != nil {
		devToolsEnabled = *options.DevToolsEnabled
	}
	C.panel_enable_devtools(p.webview, C.gboolean(boolToInt(devToolsEnabled)))

	// Set background color
	if options.Transparent {
		C.panel_set_background_color(p.webview, 0, 0, 0, 0)
	} else {
		C.panel_set_background_color(p.webview,
			C.int(options.BackgroundColour.Red),
			C.int(options.BackgroundColour.Green),
			C.int(options.BackgroundColour.Blue),
			C.int(options.BackgroundColour.Alpha),
		)
	}

	// Set zoom if specified
	if options.Zoom > 0 && options.Zoom != 1.0 {
		C.panel_set_zoom(p.webview, C.double(options.Zoom))
	}

	// Set initial visibility
	if options.Visible == nil || *options.Visible {
		C.gtk_widget_set_visible(p.fixed, 1)
	}

	// By default, the panel does NOT capture input. Clicks pass through
	// to the main webview. The frontend calls SetInputEnabled(true) when
	// the user activates the panel (e.g. by clicking inside it).
	C.gtk_widget_set_can_target(p.fixed, 0)

	// Navigate to initial URL
	if options.URL != "" {
		if len(options.Headers) > 0 {
			globalApplication.debug("[Panel-Linux] Custom headers specified (not yet supported)",
				"panelID", p.panel.id,
				"headers", options.Headers)
		}
		url := C.CString(options.URL)
		defer C.free(unsafe.Pointer(url))
		C.panel_load_url(p.webview, url)
	}

	// Open inspector if requested
	if debugMode && options.OpenInspectorOnStartup {
		C.panel_open_devtools(p.webview)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (p *linuxPanelImpl) destroy() {
	if p.fixed != nil {
		C.panel_destroy(p.fixed)
		p.fixed = nil
		p.webview = nil
	}
	// Note: the overlay is cleaned up when the window is destroyed
}

func (p *linuxPanelImpl) setBounds(bounds Rect) {
	if p.webview == nil || p.fixed == nil {
		return
	}
	C.panel_fixed_move(p.fixed, p.webview, C.int(bounds.X), C.int(bounds.Y))
	C.panel_set_size(p.webview, C.int(bounds.Width), C.int(bounds.Height))
}

func (p *linuxPanelImpl) bounds() Rect {
	if p.webview == nil {
		return Rect{}
	}
	var x, y, width, height C.int
	C.panel_get_allocation(p.webview, &x, &y, &width, &height)
	return Rect{
		X:      int(x),
		Y:      int(y),
		Width:  int(width),
		Height: int(height),
	}
}

func (p *linuxPanelImpl) setZIndex(_ int) {
	// GTK doesn't have a direct z-index concept
	// We could use gtk_box_reorder_child to change ordering
	// For now, this is a no-op
}

func (p *linuxPanelImpl) setURL(url string) {
	if p.webview == nil {
		return
	}
	urlStr := C.CString(url)
	defer C.free(unsafe.Pointer(urlStr))
	C.panel_load_url(p.webview, urlStr)
}

func (p *linuxPanelImpl) reload() {
	if p.webview == nil {
		return
	}
	C.panel_reload(p.webview)
}

func (p *linuxPanelImpl) forceReload() {
	if p.webview == nil {
		return
	}
	C.panel_force_reload(p.webview)
}

func (p *linuxPanelImpl) show() {
	if p.fixed == nil {
		return
	}
	C.gtk_widget_set_visible(p.fixed, 1)
}

func (p *linuxPanelImpl) hide() {
	if p.fixed == nil {
		return
	}
	C.gtk_widget_set_visible(p.fixed, 0)
}

func (p *linuxPanelImpl) isVisible() bool {
	if p.fixed == nil {
		return false
	}
	return C.gtk_widget_get_visible(p.fixed) != 0
}

func (p *linuxPanelImpl) setZoom(zoom float64) {
	if p.webview == nil {
		return
	}
	C.panel_set_zoom(p.webview, C.double(zoom))
}

func (p *linuxPanelImpl) getZoom() float64 {
	if p.webview == nil {
		return 1.0
	}
	return float64(C.panel_get_zoom(p.webview))
}

func (p *linuxPanelImpl) openDevTools() {
	if p.webview == nil {
		return
	}
	C.panel_open_devtools(p.webview)
}

func (p *linuxPanelImpl) focus() {
	if p.webview == nil {
		return
	}
	C.panel_focus(p.webview)
}

func (p *linuxPanelImpl) isFocused() bool {
	if p.webview == nil {
		return false
	}
	return C.panel_is_focused(p.webview) != 0
}

func (p *linuxPanelImpl) setInputEnabled(enabled bool) {
	if p.fixed == nil {
		return
	}
	val := 0
	if enabled {
		val = 1
	}
	C.gtk_widget_set_can_target(p.fixed, C.gboolean(val))
}
