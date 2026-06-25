#!/usr/bin/env python3
import os
import signal
import subprocess
import time
import gi

gi.require_version('Gtk', '3.0')
try:
    gi.require_version('AyatanaAppIndicator3', '0.1')
    from gi.repository import AyatanaAppIndicator3 as AppIndicator3
except ValueError:
    try:
        gi.require_version('AppIndicator3', '0.1')
        from gi.repository import AppIndicator3
    except ValueError:
        print("Error: python3-gi and gir1.2-ayatanaappindicator3-0.1 are required.")
        exit(1)

from gi.repository import Gtk, GLib

SERVICE_NAME = "prompttranslate.service"

class TrayIcon:
    def __init__(self):
        self.indicator = AppIndicator3.Indicator.new(
            "prompttranslate_tray",
            "accessories-dictionary",  # Built-in gnome icon
            AppIndicator3.IndicatorCategory.APPLICATION_STATUS
        )
        self.indicator.set_status(AppIndicator3.IndicatorStatus.ACTIVE)
        self.indicator.set_menu(self.build_menu())
        
        # Check service status every 2 seconds
        GLib.timeout_add_seconds(2, self.update_status)
        self.update_status()

    def build_menu(self):
        menu = Gtk.Menu()

        self.status_item = Gtk.MenuItem(label="Durum: Bilinmiyor")
        self.status_item.set_sensitive(False)
        menu.append(self.status_item)

        menu.append(Gtk.SeparatorMenuItem())

        item_start = Gtk.MenuItem(label="▶ Başlat")
        item_start.connect('activate', self.start_service)
        menu.append(item_start)

        item_stop = Gtk.MenuItem(label="⏹ Durdur")
        item_stop.connect('activate', self.stop_service)
        menu.append(item_stop)

        menu.append(Gtk.SeparatorMenuItem())

        item_quit = Gtk.MenuItem(label="Kapat (Tepsi Simgesini)")
        item_quit.connect('activate', self.quit)
        menu.append(item_quit)

        menu.show_all()
        return menu

    def is_service_running(self):
        try:
            result = subprocess.run(
                ["systemctl", "--user", "is-active", SERVICE_NAME],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            return result.stdout.strip() == "active"
        except Exception:
            return False

    def update_status(self):
        running = self.is_service_running()
        if running:
            self.status_item.set_label("Durum: 🟢 Çalışıyor")
            self.indicator.set_icon("accessories-dictionary") # Active icon
        else:
            self.status_item.set_label("Durum: 🔴 Kapalı")
            self.indicator.set_icon("process-stop") # Inactive icon
        return True # Keep running the timer

    def start_service(self, _):
        subprocess.run(["systemctl", "--user", "start", SERVICE_NAME])
        self.update_status()

    def stop_service(self, _):
        subprocess.run(["systemctl", "--user", "stop", SERVICE_NAME])
        self.update_status()

    def quit(self, _):
        Gtk.main_quit()

if __name__ == "__main__":
    signal.signal(signal.SIGINT, signal.SIG_DFL)
    TrayIcon()
    Gtk.main()
