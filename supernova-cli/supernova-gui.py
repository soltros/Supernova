#!/usr/bin/env python3

import tkinter as tk
from tkinter import ttk
import subprocess
import json
import threading
import os
import signal

# Automatically detect local binary vs system path binary
SN_CMD = "./sn" if os.path.exists("./sn") else "sn"

# --- SUPERNOVA COLOR PALETTE ---
BG_MAIN = "#120e15"
BG_SURFACE = "#1c1722"
BG_SIDEBAR = "#151119"
BG_PLAYER = "#231d2a"
ACCENT = "#d4338d"
TEXT_MAIN = "#ffffff"
TEXT_MUTED = "#8b8492"

class SupernovaGUI:
    def __init__(self, root):
        self.root = root
        self.root.title("Supernova")
        self.root.geometry("1024x768")
        self.root.configure(bg=BG_MAIN)

        self.current_process = None

        self._apply_theme()
        self._build_ui()
        self._load_sidebar()

    def _apply_theme(self):
        style = ttk.Style()
        style.theme_use("default")

        style.configure(".",
                        background=BG_MAIN,
                        foreground=TEXT_MAIN,
                        font=("Helvetica", 10),
                        troughcolor=BG_SURFACE,
                        bordercolor=BG_MAIN,
                        darkcolor=BG_MAIN,
                        lightcolor=BG_MAIN)

        style.configure("TFrame", background=BG_MAIN)
        style.configure("Sidebar.TFrame", background=BG_SIDEBAR)
        style.configure("Surface.TFrame", background=BG_SURFACE)
        style.configure("Player.TFrame", background=BG_PLAYER)

        style.configure("TButton",
                        background=ACCENT,
                        foreground=TEXT_MAIN,
                        borderwidth=0,
                        focusthickness=0,
                        padding=(15, 8),
                        font=("Helvetica", 10, "bold"))
        style.map("TButton",
                  background=[("active", "#a5276e"), ("pressed", "#7a1c51")],
                  foreground=[("active", "#ffffff")])

        style.configure("TLabel", background=BG_MAIN, foreground=TEXT_MUTED)
        style.configure("Status.TLabel", background=BG_PLAYER, foreground=TEXT_MAIN, font=("Helvetica", 11, "bold"))
        style.configure("Player.TLabel", background=BG_PLAYER, foreground=TEXT_MUTED)

        style.configure("TPanedwindow", background=BG_MAIN)
        style.configure("Sash", background=BG_MAIN, sashthickness=2)

        style.configure("Treeview",
                        background=BG_SURFACE,
                        fieldbackground=BG_SURFACE,
                        foreground=TEXT_MAIN,
                        borderwidth=0,
                        rowheight=35,
                        font=("Helvetica", 10))
        style.map("Treeview",
                  background=[("selected", ACCENT)],
                  foreground=[("selected", "#ffffff")])

        style.configure("Treeview.Heading",
                        background=BG_MAIN,
                        foreground=TEXT_MUTED,
                        borderwidth=0,
                        font=("Helvetica", 10, "bold"),
                        padding=(5, 10))
        style.map("Treeview.Heading", background=[("active", BG_SURFACE)])

    def _build_ui(self):
        content_frame = ttk.Frame(self.root)
        content_frame.pack(side=tk.TOP, fill=tk.BOTH, expand=True)

        paned = ttk.PanedWindow(content_frame, orient=tk.HORIZONTAL)
        paned.pack(side=tk.TOP, fill=tk.BOTH, expand=True, padx=10, pady=10)

        sidebar_frame = ttk.Frame(paned, width=220, style="Sidebar.TFrame")
        self.sidebar = ttk.Treeview(sidebar_frame, show="tree", selectmode="browse")
        self.sidebar.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        self.sidebar.bind("<<TreeviewSelect>>", self.on_sidebar_select)
        paned.add(sidebar_frame, weight=1)

        main_frame = ttk.Frame(paned, style="Surface.TFrame")
        columns = ("title", "artist", "album", "id")
        self.data_view = ttk.Treeview(main_frame, columns=columns, show="headings", selectmode="extended")
        self.data_view.heading("title", text="Title", anchor=tk.W)
        self.data_view.heading("artist", text="Artist", anchor=tk.W)
        self.data_view.heading("album", text="Album", anchor=tk.W)
        self.data_view.heading("id", text="ID", anchor=tk.W)

        self.data_view.column("title", width=300)
        self.data_view.column("artist", width=200)
        self.data_view.column("album", width=200)
        self.data_view.column("id", width=250, stretch=tk.NO)

        self.data_view.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)
        self.data_view.bind("<Double-1>", lambda e: self.play_selected())
        paned.add(main_frame, weight=4)

        player_frame = ttk.Frame(self.root, style="Player.TFrame", padding="15 15 15 15")
        player_frame.pack(side=tk.BOTTOM, fill=tk.X)

        self.status_var = tk.StringVar(value="Nothing Playing")
        status_label = ttk.Label(player_frame, textvariable=self.status_var, style="Status.TLabel")
        status_label.pack(side=tk.LEFT, padx=(10, 20))

        controls_frame = ttk.Frame(player_frame, style="Player.TFrame")
        controls_frame.pack(side=tk.LEFT, expand=True)

        self.play_btn = ttk.Button(controls_frame, text="Play", command=self.play_selected)
        self.play_btn.pack(side=tk.LEFT, padx=5)

        self.stop_btn = ttk.Button(controls_frame, text="Stop", command=self.stop_playback)
        self.stop_btn.pack(side=tk.LEFT, padx=5)

    def _load_sidebar(self):
        lib_node = self.sidebar.insert("", "end", text="Library", open=True)
        self.sidebar.insert(lib_node, "end", text="Tracks", tags=("nav_tracks",))
        self.sidebar.insert(lib_node, "end", text="Artists", tags=("nav_artists",))
        self.sidebar.insert(lib_node, "end", text="Albums", tags=("nav_albums",))

        user_node = self.sidebar.insert("", "end", text="User Data", open=True)
        self.sidebar.insert(user_node, "end", text="Favorites", tags=("nav_hearts",))
        self.sidebar.insert(user_node, "end", text="Playlists", tags=("nav_playlists",))

    def on_sidebar_select(self, event):
        selection = self.sidebar.selection()
        if not selection:
            return

        item = self.sidebar.item(selection[0])
        tags = item.get("tags", [])

        if "nav_tracks" in tags:
            self.load_data("tracks")
        elif "nav_artists" in tags:
            self.load_data("artists")
        elif "nav_albums" in tags:
            self.load_data("albums")
        elif "nav_playlists" in tags:
            self.load_data("playlists")
        elif "nav_hearts" in tags:
            self.load_data("hearts-details")

    def run_cli(self, args):
        cmd = [SN_CMD] + args
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, check=True)
            output = result.stdout.strip()

            # 1. Attempt to isolate and parse a JSON block
            first_brace = output.find('{')
            first_bracket = output.find('[')

            start_idx = -1
            if first_brace != -1 and first_bracket != -1:
                start_idx = min(first_brace, first_bracket)
            elif first_brace != -1:
                start_idx = first_brace
            elif first_bracket != -1:
                start_idx = first_bracket

            if start_idx != -1:
                json_str = output[start_idx:]
                try:
                    return json.loads(json_str)
                except json.JSONDecodeError:
                    pass

            return []

        except subprocess.CalledProcessError as e:
            err_msg = f"CLI Error: {e.stderr.strip()}" if e.stderr else f"CLI exited with {e.returncode}"
            print(err_msg)
            return {"error": err_msg}
        except FileNotFoundError:
            err_msg = f"Error: {SN_CMD} not found."
            print(err_msg)
            return {"error": err_msg}
        except Exception as e:
            err_msg = f"Subprocess Error: {e}"
            print(err_msg)
            return {"error": err_msg}

    def load_data(self, endpoint):
        self.status_var.set(f"Loading {endpoint}...")
        self.root.update_idletasks()

        for item in self.data_view.get_children():
            self.data_view.delete(item)

        def fetch():
            data = self.run_cli([endpoint])
            self.root.after(0, self._populate_view, endpoint, data)

        threading.Thread(target=fetch, daemon=True).start()

    def _populate_view(self, endpoint, data):
        if isinstance(data, dict) and "error" in data:
            self.status_var.set(data["error"])
            return

        if not isinstance(data, list):
            if isinstance(data, dict) and "items" in data:
                data = data["items"]
            else:
                self.status_var.set(f"Failed to parse {endpoint} data.")
                return

        for item in data:
            # Unpack hydrated nested models from hearts/details endpoint
            if isinstance(item, dict):
                if "track" in item and isinstance(item["track"], dict):
                    item = item["track"]
                elif "album" in item and isinstance(item["album"], dict):
                    item = item["album"]
                elif "artist" in item and isinstance(item["artist"], dict):
                    item = item["artist"]

            if endpoint in ("tracks", "hearts-details"):
                title = item.get("title", item.get("name", "Unknown"))
                artist = item.get("artist_name", "")
                album = item.get("album_title", "")
                item_id = item.get("id", "")
                self.data_view.insert("", "end", values=(title, artist, album, item_id))
            elif endpoint in ("artists", "albums", "playlists"):
                title = item.get("title", item.get("name", "Unknown"))
                artist = item.get("artist_name", "")
                item_id = item.get("id", "")
                self.data_view.insert("", "end", values=(title, artist, "", item_id))

        self.status_var.set("Ready")

    def play_selected(self):
        selection = self.data_view.selection()
        if not selection:
            return

        values = self.data_view.item(selection[0], "values")
        if not values:
            return

        track_id = values[3]
        title = values[0]
        artist = values[1]

        self.stop_playback()

        display_text = title
        if artist:
            display_text = f"{title} — {artist}"
        self.status_var.set(display_text)

        try:
            kwargs = {}
            if os.name == 'nt':
                kwargs['creationflags'] = subprocess.CREATE_NEW_PROCESS_GROUP
            else:
                kwargs['preexec_fn'] = os.setsid

            self.current_process = subprocess.Popen(
                [SN_CMD, "play", str(track_id)],
                stdin=subprocess.DEVNULL,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                **kwargs
            )
        except FileNotFoundError:
            self.status_var.set("Error: sn binary not found.")
            print(f"Could not locate the executable: {SN_CMD}")

    def stop_playback(self):
        if self.current_process and self.current_process.poll() is None:
            if os.name == 'nt':
                self.current_process.send_signal(signal.CTRL_BREAK_EVENT)
            else:
                os.killpg(os.getpgid(self.current_process.pid), signal.SIGTERM)
            self.current_process.wait()
            self.status_var.set("Playback Stopped")

if __name__ == "__main__":
    root = tk.Tk()
    app = SupernovaGUI(root)
    root.mainloop()
