import 'package:flutter/material.dart';
import 'package:media_kit/media_kit.dart';
import 'package:window_manager/window_manager.dart';

import 'api_client.dart';
import 'app_controller.dart';
import 'models.dart';
import 'player_controller.dart';

const ink = Color(0xFF070A12);
const panel = Color(0xFF111624);
const panel2 = Color(0xFF171D2E);
const violet = Color(0xFF8B5CF6);
const cyan = Color(0xFF22D3EE);
const pink = Color(0xFFF472B6);

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  MediaKit.ensureInitialized();
  await windowManager.ensureInitialized();
  await windowManager.waitUntilReadyToShow(
    const WindowOptions(
      size: Size(1280, 820),
      minimumSize: Size(900, 620),
      center: true,
      title: 'Supernova',
      backgroundColor: ink,
    ),
    () async {
      await windowManager.show();
      await windowManager.focus();
    },
  );

  final app = AppController();
  final player = PlayerController();
  await app.boot();
  runApp(SupernovaApp(app: app, player: player));
}

class SupernovaApp extends StatelessWidget {
  const SupernovaApp({super.key, required this.app, required this.player});
  final AppController app;
  final PlayerController player;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: app,
      builder: (context, _) => MaterialApp(
        debugShowCheckedModeBanner: false,
        title: 'Supernova',
        theme: ThemeData(
          brightness: Brightness.dark,
          useMaterial3: true,
          scaffoldBackgroundColor: ink,
          colorScheme: const ColorScheme.dark(
            primary: violet,
            secondary: cyan,
            tertiary: pink,
            surface: panel,
          ),
          inputDecorationTheme: InputDecorationTheme(
            filled: true,
            fillColor: panel2,
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(16),
              borderSide: BorderSide.none,
            ),
          ),
          cardTheme: CardThemeData(
            color: panel,
            elevation: 0,
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
          ),
        ),
        home: app.booting
            ? const Scaffold(body: Center(child: CircularProgressIndicator()))
            : app.authenticated
                ? DesktopShell(app: app, player: player)
                : AuthScreen(app: app),
      ),
    );
  }
}

class NebulaBackground extends StatelessWidget {
  const NebulaBackground({super.key});

  @override
  Widget build(BuildContext context) => DecoratedBox(
        decoration: const BoxDecoration(
          gradient: RadialGradient(
            center: Alignment(-0.65, -0.65),
            radius: 1.35,
            colors: [Color(0x553B82F6), Color(0x332A1359), ink],
          ),
        ),
        child: const DecoratedBox(
          decoration: BoxDecoration(
            gradient: RadialGradient(
              center: Alignment(0.8, 0.7),
              radius: 0.9,
              colors: [Color(0x33F472B6), Colors.transparent],
            ),
          ),
        ),
      );
}

class AuthScreen extends StatefulWidget {
  const AuthScreen({super.key, required this.app});
  final AppController app;

  @override
  State<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends State<AuthScreen> {
  final instance = TextEditingController();
  final username = TextEditingController();
  final password = TextEditingController();
  final invite = TextEditingController();
  bool register = false;
  String? localError;

  @override
  void initState() {
    super.initState();
    instance.text = widget.app.instance;
  }

  @override
  void dispose() {
    instance.dispose();
    username.dispose();
    password.dispose();
    invite.dispose();
    super.dispose();
  }

  Future<void> submit() async {
    setState(() => localError = null);
    try {
      if (register) {
        await widget.app.register(
          instanceUrl: instance.text,
          username: username.text,
          password: password.text,
          inviteCode: invite.text,
        );
      } else {
        await widget.app.login(
          instanceUrl: instance.text,
          username: username.text,
          password: password.text,
        );
      }
    } catch (error) {
      if (mounted) setState(() => localError = error.toString());
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Stack(
        children: [
          const Positioned.fill(child: NebulaBackground()),
          Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 470),
              child: Card(
                color: panel.withAlpha(240),
                child: Padding(
                  padding: const EdgeInsets.all(34),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Row(
                        children: [
                          Image.asset('assets/icon.png', width: 52, height: 52),
                          const SizedBox(width: 16),
                          const Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text('SUPERNOVA', style: TextStyle(fontSize: 25, fontWeight: FontWeight.w800, letterSpacing: 3.2)),
                              Text('YOUR MUSIC. YOUR SERVER.', style: TextStyle(fontSize: 11, color: Colors.white54, letterSpacing: 1.6)),
                            ],
                          ),
                        ],
                      ),
                      const SizedBox(height: 32),
                      TextField(
                        controller: instance,
                        keyboardType: TextInputType.url,
                        decoration: const InputDecoration(
                          labelText: 'Supernova instance',
                          hintText: 'https://music.example.com',
                          prefixIcon: Icon(Icons.public),
                        ),
                      ),
                      const SizedBox(height: 14),
                      TextField(
                        controller: username,
                        decoration: const InputDecoration(labelText: 'Username', prefixIcon: Icon(Icons.person_outline)),
                      ),
                      const SizedBox(height: 14),
                      TextField(
                        controller: password,
                        obscureText: true,
                        onSubmitted: (_) => submit(),
                        decoration: const InputDecoration(labelText: 'Password', prefixIcon: Icon(Icons.lock_outline)),
                      ),
                      if (register) ...[
                        const SizedBox(height: 14),
                        TextField(
                          controller: invite,
                          decoration: const InputDecoration(
                            labelText: 'Invite code',
                            hintText: 'Leave blank for the first account',
                            prefixIcon: Icon(Icons.key_outlined),
                          ),
                        ),
                      ],
                      if (localError != null || widget.app.error != null) ...[
                        const SizedBox(height: 16),
                        Text(localError ?? widget.app.error!, style: const TextStyle(color: Color(0xFFFCA5A5))),
                      ],
                      const SizedBox(height: 22),
                      FilledButton.icon(
                        onPressed: widget.app.busy ? null : submit,
                        icon: widget.app.busy
                            ? const SizedBox.square(dimension: 18, child: CircularProgressIndicator(strokeWidth: 2))
                            : Icon(register ? Icons.person_add_alt : Icons.login),
                        label: Padding(
                          padding: const EdgeInsets.symmetric(vertical: 14),
                          child: Text(register ? 'Create account' : 'Connect & sign in'),
                        ),
                      ),
                      const SizedBox(height: 8),
                      TextButton(
                        onPressed: widget.app.busy ? null : () => setState(() => register = !register),
                        child: Text(register ? 'Already have an account? Sign in' : 'New here? Register from the app'),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

enum Section {
  home('Home', Icons.auto_awesome_rounded),
  albums('Albums', Icons.album_rounded),
  artists('Artists', Icons.groups_2_rounded),
  search('Search', Icons.search_rounded),
  playlists('Playlists', Icons.queue_music_rounded),
  favorites('Favorites', Icons.favorite_rounded),
  podcasts('Podcasts', Icons.podcasts_rounded),
  radio('Radio', Icons.radio_rounded),
  settings('Settings', Icons.tune_rounded);

  const Section(this.label, this.icon);
  final String label;
  final IconData icon;
}

class DesktopShell extends StatefulWidget {
  const DesktopShell({super.key, required this.app, required this.player});
  final AppController app;
  final PlayerController player;

  @override
  State<DesktopShell> createState() => _DesktopShellState();
}

class _DesktopShellState extends State<DesktopShell> {
  Section section = Section.home;

  Widget page() => switch (section) {
        Section.home => HomePage(api: widget.app.api, player: widget.player),
        Section.albums => AlbumsPage(api: widget.app.api, player: widget.player),
        Section.artists => ArtistsPage(api: widget.app.api, player: widget.player),
        Section.search => SearchPage(api: widget.app.api, player: widget.player),
        Section.playlists => PlaylistsPage(api: widget.app.api, player: widget.player),
        Section.favorites => FavoritesPage(api: widget.app.api, player: widget.player),
        Section.podcasts => PodcastsPage(api: widget.app.api, player: widget.player),
        Section.radio => RadioPage(api: widget.app.api, player: widget.player),
        Section.settings => SettingsPage(app: widget.app),
      };

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Stack(
        children: [
          const Positioned.fill(child: NebulaBackground()),
          Column(
            children: [
              Expanded(
                child: Row(
                  children: [
                    Container(
                      width: 220,
                      margin: const EdgeInsets.fromLTRB(14, 14, 0, 8),
                      decoration: BoxDecoration(
                        color: panel.withAlpha(235),
                        borderRadius: BorderRadius.circular(24),
                        border: Border.all(color: Colors.white.withAlpha(14)),
                      ),
                      child: Column(
                        children: [
                          Padding(
                            padding: const EdgeInsets.fromLTRB(20, 24, 16, 20),
                            child: Row(
                              children: [
                                Image.asset('assets/icon.png', width: 36, height: 36),
                                const SizedBox(width: 12),
                                const Text('SUPERNOVA', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w800, letterSpacing: 2)),
                              ],
                            ),
                          ),
                          Expanded(
                            child: ListView(
                              padding: const EdgeInsets.symmetric(horizontal: 10),
                              children: [
                                for (final item in Section.values)
                                  Padding(
                                    padding: const EdgeInsets.only(bottom: 4),
                                    child: ListTile(
                                      selected: section == item,
                                      selectedTileColor: violet.withAlpha(32),
                                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                                      leading: Icon(item.icon),
                                      title: Text(item.label),
                                      onTap: () => setState(() => section = item),
                                    ),
                                  ),
                              ],
                            ),
                          ),
                          Padding(
                            padding: const EdgeInsets.all(16),
                            child: Row(
                              children: [
                                const CircleAvatar(radius: 16, backgroundColor: violet, child: Icon(Icons.person, size: 18)),
                                const SizedBox(width: 10),
                                Expanded(
                                  child: Column(
                                    crossAxisAlignment: CrossAxisAlignment.start,
                                    children: [
                                      Text(widget.app.user?.username ?? '', overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w600)),
                                      Text(widget.app.user?.isAdmin == true ? 'Administrator' : 'Listener', style: const TextStyle(fontSize: 11, color: Colors.white54)),
                                    ],
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                    Expanded(
                      child: Padding(
                        padding: const EdgeInsets.fromLTRB(18, 14, 14, 8),
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(24),
                          child: ColoredBox(color: panel.withAlpha(220), child: page()),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              PlayerBar(api: widget.app.api, player: widget.player),
            ],
          ),
        ],
      ),
    );
  }
}

class PageFrame extends StatelessWidget {
  const PageFrame({super.key, required this.title, this.subtitle, required this.child, this.actions = const []});
  final String title;
  final String? subtitle;
  final Widget child;
  final List<Widget> actions;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.all(28),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(title, style: Theme.of(context).textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w800)),
                      if (subtitle != null)
                        Padding(padding: const EdgeInsets.only(top: 4), child: Text(subtitle!, style: const TextStyle(color: Colors.white54))),
                    ],
                  ),
                ),
                ...actions,
              ],
            ),
            const SizedBox(height: 24),
            Expanded(child: child),
          ],
        ),
      );
}

class HomePage extends StatelessWidget {
  const HomePage({super.key, required this.api, required this.player});
  final ApiClient api;
  final PlayerController player;

  @override
  Widget build(BuildContext context) => PageFrame(
        title: 'Good listening.',
        subtitle: 'Your Supernova instance, without the browser chrome.',
        child: FutureBuilder<DashboardData>(
          future: api.dashboard(),
          builder: (context, snapshot) {
            if (snapshot.hasError) return ErrorView(snapshot.error);
            if (!snapshot.hasData) return const Center(child: CircularProgressIndicator());
            final data = snapshot.data!;
            return ListView(
              children: [
                const SectionHeader('Recently added'),
                AlbumStrip(api: api, albums: data.recentAlbums, player: player),
                const SizedBox(height: 26),
                const SectionHeader('Recently played'),
                TrackList(api: api, tracks: data.recentTracks, player: player),
                const SizedBox(height: 26),
                const SectionHeader('Favorites'),
                TrackList(api: api, tracks: data.favoriteTracks, player: player),
              ],
            );
          },
        ),
      );
}

class AlbumsPage extends StatelessWidget {
  const AlbumsPage({super.key, required this.api, required this.player});
  final ApiClient api;
  final PlayerController player;

  @override
  Widget build(BuildContext context) => PageFrame(
        title: 'Albums',
        subtitle: 'Everything on your server.',
        child: FutureBuilder<List<Album>>(
          future: api.allAlbums(),
          builder: (context, snapshot) {
            if (snapshot.hasError) return ErrorView(snapshot.error);
            if (!snapshot.hasData) return const Center(child: CircularProgressIndicator());
            final albums = snapshot.data!;
            return LayoutBuilder(
              builder: (context, constraints) {
                final columns = (constraints.maxWidth / 190).floor().clamp(2, 7);
                return GridView.builder(
                  gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                    crossAxisCount: columns,
                    childAspectRatio: .76,
                    crossAxisSpacing: 14,
                    mainAxisSpacing: 14,
                  ),
                  itemCount: albums.length,
                  itemBuilder: (context, index) => AlbumCard(api: api, album: albums[index], player: player),
                );
              },
            );
          },
        ),
      );
}

class AlbumCard extends StatelessWidget {
  const AlbumCard({super.key, required this.api, required this.album, required this.player});
  final ApiClient api;
  final Album album;
  final PlayerController player;

  @override
  Widget build(BuildContext context) => Card(
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: () => showAlbum(context, api, album, player),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Expanded(child: Artwork(url: api.albumArtUrl(album.id))),
              Padding(
                padding: const EdgeInsets.fromLTRB(14, 12, 14, 2),
                child: Text(album.title, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w700)),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(14, 0, 14, 14),
                child: Text(
                  album.artistName ?? (album.releaseYear > 0 ? '${album.releaseYear}' : 'Album'),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(color: Colors.white54, fontSize: 12),
                ),
              ),
            ],
          ),
        ),
      );
}

Future<void> showAlbum(BuildContext context, ApiClient api, Album album, PlayerController player) async {
  await showDialog<void>(
    context: context,
    builder: (context) => Dialog(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 760, maxHeight: 680),
        child: FutureBuilder<List<Track>>(
          future: api.allTracks(albumId: album.id),
          builder: (context, snapshot) => Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              children: [
                Row(
                  children: [
                    SizedBox(width: 86, height: 86, child: Artwork(url: api.albumArtUrl(album.id))),
                    const SizedBox(width: 18),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(album.title, style: Theme.of(context).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w800)),
                          Text(album.artistName ?? '', style: const TextStyle(color: Colors.white54)),
                        ],
                      ),
                    ),
                    IconButton(onPressed: () => Navigator.pop(context), icon: const Icon(Icons.close)),
                  ],
                ),
                const SizedBox(height: 18),
                if (snapshot.hasError)
                  Expanded(child: ErrorView(snapshot.error))
                else if (!snapshot.hasData)
                  const Expanded(child: Center(child: CircularProgressIndicator()))
                else
                  Expanded(child: TrackList(api: api, tracks: snapshot.data!, player: player)),
              ],
            ),
          ),
        ),
      ),
    ),
  );
}

class ArtistsPage extends StatelessWidget {
  const ArtistsPage({super.key, required this.api, required this.player});
  final ApiClient api;
  final PlayerController player;

  @override
  Widget build(BuildContext context) => PageFrame(
        title: 'Artists',
        child: FutureBuilder<List<Artist>>(
          future: api.allArtists(),
          builder: (context, snapshot) {
            if (snapshot.hasError) return ErrorView(snapshot.error);
            if (!snapshot.hasData) return const Center(child: CircularProgressIndicator());
            return ListView.separated(
              itemCount: snapshot.data!.length,
              separatorBuilder: (_, _) => const Divider(height: 1),
              itemBuilder: (context, index) {
                final artist = snapshot.data![index];
                return ListTile(
                  leading: CircleAvatar(
                    backgroundColor: violet.withAlpha(45),
                    child: Text(artist.name.isEmpty ? '?' : artist.name[0].toUpperCase()),
                  ),
                  title: Text(artist.name),
                  subtitle: artist.bio?.isNotEmpty == true
                      ? Text(artist.bio!, maxLines: 1, overflow: TextOverflow.ellipsis)
                      : null,
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () async {
                    final albums = await api.allAlbums(artistId: artist.id);
                    if (!context.mounted) return;
                    await showDialog<void>(
                      context: context,
                      builder: (context) => Dialog(
                        child: SizedBox(
                          width: 820,
                          height: 620,
                          child: Padding(
                            padding: const EdgeInsets.all(24),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Row(
                                  children: [
                                    Expanded(child: Text(artist.name, style: Theme.of(context).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w800))),
                                    IconButton(onPressed: () => Navigator.pop(context), icon: const Icon(Icons.close)),
                                  ],
                                ),
                                const SizedBox(height: 16),
                                Expanded(child: AlbumStrip(api: api, albums: albums, player: player, grid: true)),
                              ],
                            ),
                          ),
                        ),
                      ),
                    );
                  },
                );
              },
            );
          },
        ),
      );
}

class SearchPage extends StatefulWidget {
  const SearchPage({super.key, required this.api, required this.player});
  final ApiClient api;
  final PlayerController player;

  @override
  State<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends State<SearchPage> {
  final controller = TextEditingController();
  Future<SearchResults>? future;

  @override
  void dispose() {
    controller.dispose();
    super.dispose();
  }

  void runSearch() {
    final query = controller.text.trim();
    if (query.isNotEmpty) setState(() => future = widget.api.search(query));
  }

  @override
  Widget build(BuildContext context) => PageFrame(
        title: 'Search',
        child: Column(
          children: [
            TextField(
              controller: controller,
              autofocus: true,
              onSubmitted: (_) => runSearch(),
              decoration: InputDecoration(
                hintText: 'Artists, albums, tracks…',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: IconButton(onPressed: runSearch, icon: const Icon(Icons.arrow_forward)),
              ),
            ),
            const SizedBox(height: 20),
            Expanded(
              child: future == null
                  ? const Center(child: Text('Search your library', style: TextStyle(color: Colors.white38)))
                  : FutureBuilder<SearchResults>(
                      future: future,
                      builder: (context, snapshot) {
                        if (snapshot.hasError) return ErrorView(snapshot.error);
                        if (!snapshot.hasData) return const Center(child: CircularProgressIndicator());
                        final data = snapshot.data!;
                        return ListView(
                          children: [
                            if (data.artists.isNotEmpty) ...[
                              const SectionHeader('Artists'),
                              ...data.artists.map((artist) => ListTile(leading: const Icon(Icons.person), title: Text(artist.name))),
                            ],
                            if (data.albums.isNotEmpty) ...[
                              const SizedBox(height: 16),
                              const SectionHeader('Albums'),
                              SizedBox(height: 230, child: AlbumStrip(api: widget.api, albums: data.albums, player: widget.player)),
                            ],
                            if (data.tracks.isNotEmpty) ...[
                              const SizedBox(height: 16),
                              const SectionHeader('Tracks'),
                              TrackList(api: widget.api, tracks: data.tracks, player: widget.player),
                            ],
                          ],
                        );
                      },
                    ),
            ),
          ],
        ),
      );
}

class PlaylistsPage extends StatefulWidget {
  const PlaylistsPage({super.key, required this.api, required this.player});
  final ApiClient api;
  final PlayerController player;

  @override
  State<PlaylistsPage> createState() => _PlaylistsPageState();
}

class _PlaylistsPageState extends State<PlaylistsPage> {
  late Future<List<Playlist>> future = widget.api.playlists();

  void refresh() => setState(() => future = widget.api.playlists());

  Future<void> createPlaylist() async {
    final input = TextEditingController();
    final name = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('New playlist'),
        content: TextField(controller: input, autofocus: true, decoration: const InputDecoration(labelText: 'Name')),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(context, input.text.trim()), child: const Text('Create')),
        ],
      ),
    );
    input.dispose();
    if (name != null && name.isNotEmpty) {
      await widget.api.createPlaylist(name);
      refresh();
    }
  }

  @override
  Widget build(BuildContext context) => PageFrame(
        title: 'Playlists',
        actions: [FilledButton.icon(onPressed: createPlaylist, icon: const Icon(Icons.add), label: const Text('New playlist'))],
        child: FutureBuilder<List<Playlist>>(
          future: future,
          builder: (context, snapshot) {
            if (snapshot.hasError) return ErrorView(snapshot.error);
            if (!snapshot.hasData) return const Center(child: CircularProgressIndicator());
            return ListView.builder(
              itemCount: snapshot.data!.length,
              itemBuilder: (context, index) {
                final playlist = snapshot.data![index];
                return ListTile(
                  leading: const CircleAvatar(child: Icon(Icons.queue_music)),
                  title: Text(playlist.name),
                  trailing: IconButton(
                    icon: const Icon(Icons.delete_outline),
                    onPressed: () async {
                      await widget.api.deletePlaylist(playlist.id);
                      refresh();
                    },
                  ),
                  onTap: () async {
                    final tracks = await widget.api.playlistTracks(playlist.id);
                    if (!context.mounted) return;
                    await showDialog<void>(
                      context: context,
                      builder: (context) => Dialog(
                        child: SizedBox(
                          width: 760,
                          height: 620,
                          child: Padding(
                            padding: const EdgeInsets.all(24),
                            child: Column(
                              children: [
                                Row(
                                  children: [
                                    Expanded(child: Text(playlist.name, style: Theme.of(context).textTheme.headlineSmall)),
                                    IconButton(onPressed: () => Navigator.pop(context), icon: const Icon(Icons.close)),
                                  ],
                                ),
                                const SizedBox(height: 12),
                                Expanded(child: TrackList(api: widget.api, tracks: tracks, player: widget.player)),
                              ],
                            ),
                          ),
                        ),
                      ),
                    );
                  },
                );
              },
            );
          },
        ),
      );
}

class FavoritesPage extends StatelessWidget {
  const FavoritesPage({super.key, required this.api, required this.player});
  final ApiClient api;
  final PlayerController player;

  @override
  Widget build(BuildContext context) => PageFrame(
        title: 'Favorites',
        child: FutureBuilder<HeartDetails>(
          future: api.heartDetails(),
          builder: (context, snapshot) {
            if (snapshot.hasError) return ErrorView(snapshot.error);
            if (!snapshot.hasData) return const Center(child: CircularProgressIndicator());
            final data = snapshot.data!;
            return ListView(
              children: [
                if (data.albums.isNotEmpty) ...[
                  const SectionHeader('Albums'),
                  SizedBox(height: 230, child: AlbumStrip(api: api, albums: data.albums, player: player)),
                  const SizedBox(height: 22),
                ],
                const SectionHeader('Tracks'),
                TrackList(api: api, tracks: data.tracks, player: player),
              ],
            );
          },
        ),
      );
}

class PodcastsPage extends StatefulWidget {
  const PodcastsPage({super.key, required this.api, required this.player});
  final ApiClient api;
  final PlayerController player;

  @override
  State<PodcastsPage> createState() => _PodcastsPageState();
}

class _PodcastsPageState extends State<PodcastsPage> {
  late Future<List<Map<String, dynamic>>> subscriptions = widget.api.podcastSubscriptions();
  final search = TextEditingController();
  Future<List<Map<String, dynamic>>>? results;

  @override
  void dispose() {
    search.dispose();
    super.dispose();
  }

  void refresh() => setState(() => subscriptions = widget.api.podcastSubscriptions());

  Future<void> showEpisodes(Map<String, dynamic> feed) async {
    final id = (feed['feed_id'] ?? feed['id'] ?? '').toString();
    if (id.isEmpty) return;
    final episodes = await widget.api.podcastEpisodes(id);
    if (!mounted) return;
    await showDialog<void>(
      context: context,
      builder: (context) => Dialog(
        child: SizedBox(
          width: 820,
          height: 660,
          child: Padding(
            padding: const EdgeInsets.all(22),
            child: Column(
              children: [
                Row(
                  children: [
                    Expanded(child: Text((feed['title'] ?? 'Podcast').toString(), style: Theme.of(context).textTheme.headlineSmall)),
                    IconButton(onPressed: () => Navigator.pop(context), icon: const Icon(Icons.close)),
                  ],
                ),
                Expanded(
                  child: ListView.builder(
                    itemCount: episodes.length,
                    itemBuilder: (context, index) {
                      final episode = episodes[index];
                      final title = (episode['title'] ?? 'Episode').toString();
                      final url = (episode['enclosureUrl'] ?? episode['enclosure_url'] ?? '').toString();
                      return ListTile(
                        leading: const Icon(Icons.play_circle_outline),
                        title: Text(title),
                        subtitle: Text((episode['datePublishedPretty'] ?? episode['description'] ?? '').toString(), maxLines: 1, overflow: TextOverflow.ellipsis),
                        onTap: url.isEmpty ? null : () => widget.player.playExternal(url, title, subtitle: (feed['title'] ?? '').toString()),
                      );
                    },
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) => PageFrame(
        title: 'Podcasts',
        subtitle: 'Subscriptions stay with your Supernova account.',
        child: ListView(
          children: [
            TextField(
              controller: search,
              onSubmitted: (query) => setState(() => results = widget.api.podcastSearch(query)),
              decoration: InputDecoration(
                hintText: 'Find a podcast',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: IconButton(
                  icon: const Icon(Icons.arrow_forward),
                  onPressed: () {
                    final query = search.text.trim();
                    if (query.isNotEmpty) setState(() => results = widget.api.podcastSearch(query));
                  },
                ),
              ),
            ),
            if (results != null) ...[
              const SizedBox(height: 18),
              FutureBuilder<List<Map<String, dynamic>>>(
                future: results,
                builder: (context, snapshot) {
                  if (!snapshot.hasData) return const LinearProgressIndicator();
                  return Column(
                    children: snapshot.data!.take(10).map((feed) => ListTile(
                          leading: const Icon(Icons.podcasts),
                          title: Text((feed['title'] ?? 'Podcast').toString()),
                          subtitle: Text((feed['author'] ?? '').toString()),
                          trailing: IconButton(
                            icon: const Icon(Icons.add_circle_outline),
                            onPressed: () async {
                              await widget.api.subscribePodcast(feed);
                              refresh();
                            },
                          ),
                          onTap: () => showEpisodes(feed),
                        )).toList(),
                  );
                },
              ),
            ],
            const SizedBox(height: 24),
            const SectionHeader('Subscriptions'),
            FutureBuilder<List<Map<String, dynamic>>>(
              future: subscriptions,
              builder: (context, snapshot) {
                if (snapshot.hasError) return ErrorView(snapshot.error);
                if (!snapshot.hasData) return const LinearProgressIndicator();
                return Column(
                  children: snapshot.data!.map((feed) => ListTile(
                        leading: const CircleAvatar(child: Icon(Icons.podcasts)),
                        title: Text((feed['title'] ?? 'Podcast').toString()),
                        trailing: IconButton(
                          icon: const Icon(Icons.remove_circle_outline),
                          onPressed: () async {
                            await widget.api.unsubscribePodcast((feed['feed_id'] ?? '').toString());
                            refresh();
                          },
                        ),
                        onTap: () => showEpisodes(feed),
                      )).toList(),
                );
              },
            ),
          ],
        ),
      );
}

class RadioPage extends StatefulWidget {
  const RadioPage({super.key, required this.api, required this.player});
  final ApiClient api;
  final PlayerController player;

  @override
  State<RadioPage> createState() => _RadioPageState();
}

class _RadioPageState extends State<RadioPage> {
  late Future<List<Map<String, dynamic>>> subscriptions = widget.api.radioSubscriptions();
  final search = TextEditingController();
  Future<List<Map<String, dynamic>>>? results;

  @override
  void dispose() {
    search.dispose();
    super.dispose();
  }

  void refresh() => setState(() => subscriptions = widget.api.radioSubscriptions());

  Future<void> play(Map<String, dynamic> station) {
    final url = (station['url_resolved'] ?? station['url'] ?? '').toString();
    return widget.player.playExternal(url, (station['name'] ?? 'Radio').toString(), subtitle: 'Internet radio');
  }

  Widget stationRow(Map<String, dynamic> station, {required bool saved}) => ListTile(
        leading: IconButton(icon: const Icon(Icons.play_circle_fill), onPressed: () => play(station)),
        title: Text((station['name'] ?? 'Radio station').toString()),
        subtitle: Text((station['country'] ?? station['codec'] ?? '').toString()),
        trailing: IconButton(
          icon: Icon(saved ? Icons.remove_circle_outline : Icons.add_circle_outline),
          onPressed: () async {
            if (saved) {
              await widget.api.unsubscribeRadio((station['station_id'] ?? station['stationuuid'] ?? '').toString());
            } else {
              await widget.api.subscribeRadio(station);
            }
            refresh();
          },
        ),
        onTap: () => play(station),
      );

  @override
  Widget build(BuildContext context) => PageFrame(
        title: 'Radio',
        subtitle: 'Radio-Browser stations, played natively.',
        child: ListView(
          children: [
            TextField(
              controller: search,
              onSubmitted: (query) => setState(() => results = widget.api.radioSearch(query)),
              decoration: InputDecoration(
                hintText: 'Search stations',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: IconButton(
                  icon: const Icon(Icons.arrow_forward),
                  onPressed: () {
                    final query = search.text.trim();
                    if (query.isNotEmpty) setState(() => results = widget.api.radioSearch(query));
                  },
                ),
              ),
            ),
            if (results != null) ...[
              const SizedBox(height: 18),
              FutureBuilder<List<Map<String, dynamic>>>(
                future: results,
                builder: (context, snapshot) {
                  if (!snapshot.hasData) return const LinearProgressIndicator();
                  return Column(children: snapshot.data!.take(12).map((station) => stationRow(station, saved: false)).toList());
                },
              ),
            ],
            const SizedBox(height: 24),
            const SectionHeader('Saved stations'),
            FutureBuilder<List<Map<String, dynamic>>>(
              future: subscriptions,
              builder: (context, snapshot) {
                if (snapshot.hasError) return ErrorView(snapshot.error);
                if (!snapshot.hasData) return const LinearProgressIndicator();
                return Column(children: snapshot.data!.map((station) => stationRow(station, saved: true)).toList());
              },
            ),
          ],
        ),
      );
}

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key, required this.app});
  final AppController app;

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  late Future<List<Map<String, dynamic>>> plugins = widget.app.api.plugins();
  String? status;

  @override
  Widget build(BuildContext context) => PageFrame(
        title: 'Settings',
        subtitle: widget.app.instance,
        child: ListView(
          children: [
            Card(
              child: ListTile(
                leading: const CircleAvatar(backgroundColor: violet, child: Icon(Icons.person)),
                title: Text(widget.app.user?.username ?? ''),
                subtitle: Text(widget.app.user?.isAdmin == true ? 'Administrator' : 'Listener'),
                trailing: FilledButton.tonalIcon(
                  onPressed: widget.app.logout,
                  icon: const Icon(Icons.logout),
                  label: const Text('Sign out'),
                ),
              ),
            ),
            if (widget.app.user?.isAdmin == true) ...[
              const SizedBox(height: 18),
              const SectionHeader('Server'),
              Wrap(
                spacing: 10,
                runSpacing: 10,
                children: [
                  FilledButton.tonalIcon(
                    onPressed: () async {
                      await widget.app.api.scanLibrary();
                      if (mounted) setState(() => status = 'Library scan started.');
                    },
                    icon: const Icon(Icons.sync),
                    label: const Text('Scan library'),
                  ),
                  FilledButton.tonalIcon(
                    onPressed: () async {
                      await widget.app.api.resetArtists();
                      if (mounted) setState(() => status = 'Artist enrichment reset.');
                    },
                    icon: const Icon(Icons.refresh),
                    label: const Text('Reset artist enrichment'),
                  ),
                ],
              ),
              if (status != null) Padding(padding: const EdgeInsets.only(top: 12), child: Text(status!)),
            ],
            const SizedBox(height: 24),
            const SectionHeader('Plugins'),
            FutureBuilder<List<Map<String, dynamic>>>(
              future: plugins,
              builder: (context, snapshot) {
                if (!snapshot.hasData) return const LinearProgressIndicator();
                return Column(
                  children: snapshot.data!.map((plugin) {
                    final enabled = plugin['enabled'] == true;
                    return ListTile(
                      leading: Icon(enabled ? Icons.extension : Icons.extension_off),
                      title: Text((plugin['name'] ?? plugin['id'] ?? 'Plugin').toString()),
                      subtitle: Text((plugin['description'] ?? '').toString()),
                      trailing: enabled
                          ? const Icon(Icons.check_circle, color: Color(0xFF86EFAC))
                          : const Text('Disabled', style: TextStyle(color: Colors.white38)),
                    );
                  }).toList(),
                );
              },
            ),
            const SizedBox(height: 24),
            OutlinedButton.icon(
              onPressed: widget.app.forgetInstance,
              icon: const Icon(Icons.delete_outline),
              label: const Text('Forget this instance'),
            ),
          ],
        ),
      );
}

class TrackList extends StatelessWidget {
  const TrackList({super.key, required this.api, required this.tracks, required this.player});
  final ApiClient api;
  final List<Track> tracks;
  final PlayerController player;

  @override
  Widget build(BuildContext context) {
    if (tracks.isEmpty) {
      return const Padding(
        padding: EdgeInsets.all(16),
        child: Text('Nothing here yet.', style: TextStyle(color: Colors.white38)),
      );
    }
    return Column(
      children: [
        for (var index = 0; index < tracks.length; index++)
          ListTile(
            dense: true,
            leading: SizedBox(
              width: 32,
              child: Center(
                child: Text(
                  tracks[index].trackNumber > 0 ? '${tracks[index].trackNumber}' : '${index + 1}',
                  style: const TextStyle(color: Colors.white38),
                ),
              ),
            ),
            title: Text(tracks[index].title),
            subtitle: Text(
              [tracks[index].artistName, tracks[index].albumTitle]
                  .where((value) => value != null && value!.isNotEmpty)
                  .join(' • '),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: Text(formatDuration(tracks[index].duration), style: const TextStyle(color: Colors.white38)),
            onTap: () => player.playTrack(api, tracks[index], context: tracks),
          ),
      ],
    );
  }
}

class AlbumStrip extends StatelessWidget {
  const AlbumStrip({super.key, required this.api, required this.albums, required this.player, this.grid = false});
  final ApiClient api;
  final List<Album> albums;
  final PlayerController player;
  final bool grid;

  @override
  Widget build(BuildContext context) {
    if (albums.isEmpty) return const SizedBox.shrink();
    if (grid) {
      return GridView.builder(
        gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
          maxCrossAxisExtent: 190,
          childAspectRatio: .76,
          crossAxisSpacing: 14,
          mainAxisSpacing: 14,
        ),
        itemCount: albums.length,
        itemBuilder: (context, index) => AlbumCard(api: api, album: albums[index], player: player),
      );
    }
    return SizedBox(
      height: 230,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        itemCount: albums.length,
        separatorBuilder: (_, _) => const SizedBox(width: 12),
        itemBuilder: (context, index) => SizedBox(
          width: 160,
          child: AlbumCard(api: api, album: albums[index], player: player),
        ),
      ),
    );
  }
}

class Artwork extends StatelessWidget {
  const Artwork({super.key, required this.url});
  final String url;

  @override
  Widget build(BuildContext context) => Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [Color(0xFF312E81), Color(0xFF164E63), Color(0xFF4C1D95)],
          ),
        ),
        child: Image.network(
          url,
          fit: BoxFit.cover,
          errorBuilder: (_, _, _) => const Center(
            child: Icon(Icons.graphic_eq_rounded, size: 54, color: Colors.white24),
          ),
        ),
      );
}

class SectionHeader extends StatelessWidget {
  const SectionHeader(this.text, {super.key});
  final String text;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.only(bottom: 12),
        child: Text(text, style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800)),
      );
}

class ErrorView extends StatelessWidget {
  const ErrorView(this.error, {super.key});
  final Object? error;

  @override
  Widget build(BuildContext context) => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            error?.toString() ?? 'Something went wrong',
            textAlign: TextAlign.center,
            style: const TextStyle(color: Color(0xFFFCA5A5)),
          ),
        ),
      );
}

class PlayerBar extends StatelessWidget {
  const PlayerBar({super.key, required this.api, required this.player});
  final ApiClient api;
  final PlayerController player;

  @override
  Widget build(BuildContext context) => AnimatedBuilder(
        animation: player,
        builder: (context, _) {
          final maxMs = player.duration.inMilliseconds <= 0 ? 1 : player.duration.inMilliseconds;
          final posMs = player.position.inMilliseconds.clamp(0, maxMs);
          return Container(
            height: 98,
            margin: const EdgeInsets.fromLTRB(14, 0, 14, 14),
            padding: const EdgeInsets.symmetric(horizontal: 22),
            decoration: BoxDecoration(
              color: const Color(0xF20D1220),
              borderRadius: BorderRadius.circular(24),
              border: Border.all(color: Colors.white.withAlpha(15)),
            ),
            child: Row(
              children: [
                Container(
                  width: 58,
                  height: 58,
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(14),
                    gradient: const LinearGradient(colors: [violet, cyan]),
                  ),
                  child: const Icon(Icons.graphic_eq, size: 30),
                ),
                const SizedBox(width: 15),
                SizedBox(
                  width: 260,
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(player.title, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w700)),
                      Text(player.subtitle, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(color: Colors.white54, fontSize: 12)),
                    ],
                  ),
                ),
                IconButton(onPressed: () => player.previous(api), icon: const Icon(Icons.skip_previous_rounded)),
                IconButton.filled(
                  onPressed: player.title == 'Nothing playing' ? null : player.toggle,
                  icon: Icon(player.playing ? Icons.pause_rounded : Icons.play_arrow_rounded),
                ),
                IconButton(onPressed: () => player.next(api), icon: const Icon(Icons.skip_next_rounded)),
                const SizedBox(width: 16),
                Text(formatDuration(player.position), style: const TextStyle(fontSize: 11, color: Colors.white54)),
                Expanded(
                  child: Slider(
                    value: posMs.toDouble(),
                    max: maxMs.toDouble(),
                    onChanged: player.title == 'Nothing playing'
                        ? null
                        : (value) => player.seek(Duration(milliseconds: value.round())),
                  ),
                ),
                Text(formatDuration(player.duration), style: const TextStyle(fontSize: 11, color: Colors.white54)),
              ],
            ),
          );
        },
      );
}

String formatDuration(Duration value) {
  final minutes = value.inMinutes;
  final seconds = value.inSeconds.remainder(60).toString().padLeft(2, '0');
  return '$minutes:$seconds';
}
