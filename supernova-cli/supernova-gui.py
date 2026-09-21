#!/usr/bin/env python3

import tkinter as tk
from tkinter import ttk
import subprocess
import json
import threading
import os
import signal

# Prefer the CLI shipped next to this GUI, independent of the caller's CWD.
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
BUNDLED_SN = os.path.join(SCRIPT_DIR, "sn")
SN_CMD = BUNDLED_SN if os.path.isfile(BUNDLED_SN) else (shutil.which("sn") or "sn")

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
        self.request_generation += 1
        generation = self.request_generation
        self.status_var.set(f"Loading {endpoint}...")
        self.root.update_idletasks()

        for item in self.data_view.get_children():
            self.data_view.delete(item)

        def fetch():
            data = self.run_cli([endpoint])
            self.root.after(0, self._populate_view, generation, endpoint, data)

        threading.Thread(target=fetch, daemon=True).start()

    def _normalize_rows(self, endpoint, data):
        if endpoint == "hearts-details":
            if not isinstance(data, dict):
                return None
            rows = []
            for key, entity_type in (
                ("tracks", "track"),
                ("albums", "album"),
                ("artists", "artist"),
                ("playlists", "playlist"),
            ):
                values = data.get(key, [])
                if not isinstance(values, list):
                    return None
                for value in values:
                    if isinstance(value, dict):
                        rows.append((entity_type, value))
            return rows

        if isinstance(data, dict) and "items" in data and isinstance(data["items"], list):
            data = data["items"]
        if not isinstance(data, list):
            return None

        entity_type = {
            "tracks": "track",
            "artists": "artist",
            "albums": "album",
            "playlists": "playlist",
        }.get(endpoint, endpoint)
        return [(entity_type, item) for item in data if isinstance(item, dict)]

    def _populate_view(self, generation, endpoint, data):
        if generation != self.request_generation:
            return
        if isinstance(data, dict) and "error" in data:
            self.status_var.set(data["error"])
            return

        rows = self._normalize_rows(endpoint, data)
        if rows is None:
            self.status_var.set(f"Failed to parse {endpoint} data.")
            return

        for entity_type, item in rows:
            title = item.get("title", item.get("name", "Unknown"))
            artist = item.get("artist_name", "")
            album = item.get("album_title", "")
            item_id = item.get("id", "")
            self.data_view.insert(
                "",
                "end",
                values=(title, artist, album, item_id),
                tags=(entity_type,),
            )

        self.status_var.set("Ready")

    def play_selected(self):
        selection = self.data_view.selection()
        if not selection:
            return

        row = self.data_view.item(selection[0])
        values = row.get("values", ())
        if not values:
            return
        if "track" not in row.get("tags", ()):
            self.status_var.set("Select a track to start playback.")
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
                kwargs['start_new_session'] = True

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
        process = self.current_process
        self.current_process = None
        if not process or process.poll() is not None:
            return

        try:
            if os.name == 'nt':
                process.send_signal(signal.CTRL_BREAK_EVENT)
            else:
                os.killpg(process.pid, signal.SIGTERM)
            process.wait(timeout=3)
        except (subprocess.TimeoutExpired, ProcessLookupError):
            try:
                if os.name == 'nt':
                    process.kill()
                else:
                    os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            try:
                process.wait(timeout=1)
            except subprocess.TimeoutExpired:
                pass
        finally:
            self.status_var.set("Playback Stopped")

if __name__ == "__main__":
    root = tk.Tk()
    app = SupernovaGUI(root)
    root.mainloop()
