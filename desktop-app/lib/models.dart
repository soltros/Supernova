class User {
  const User({required this.id, required this.username, this.isAdmin = false});
  final String id;
  final String username;
  final bool isAdmin;

  factory User.fromJson(Map<String, dynamic> json) => User(
        id: json['id']?.toString() ?? '',
        username: json['username']?.toString() ?? '',
        isAdmin: json['is_admin'] == true || json['is_admin'] == 1,
      );
}

class Album {
  const Album({
    required this.id,
    required this.title,
    this.releaseYear = 0,
    this.artistId,
    this.artistName,
    this.bio,
  });
  final String id;
  final String title;
  final int releaseYear;
  final String? artistId;
  final String? artistName;
  final String? bio;

  factory Album.fromJson(Map<String, dynamic> json) => Album(
        id: json['id']?.toString() ?? '',
        title: json['title']?.toString() ?? 'Untitled album',
        releaseYear: (json['release_year'] as num?)?.toInt() ?? 0,
        artistId: json['artist_id']?.toString(),
        artistName: json['artist_name']?.toString(),
        bio: json['bio']?.toString(),
      );
}

class Artist {
  const Artist({required this.id, required this.name, this.imageUrl, this.bio});
  final String id;
  final String name;
  final String? imageUrl;
  final String? bio;

  factory Artist.fromJson(Map<String, dynamic> json) => Artist(
        id: json['id']?.toString() ?? '',
        name: json['name']?.toString() ?? 'Unknown artist',
        imageUrl: json['image_url']?.toString(),
        bio: json['bio']?.toString(),
      );
}

class Track {
  const Track({
    required this.id,
    required this.albumId,
    required this.title,
    this.trackNumber = 0,
    this.discNumber = 0,
    this.durationMs = 0,
    this.format = '',
    this.bitrate = 0,
    this.artistId,
    this.artistName,
    this.albumTitle,
  });

  final String id;
  final String albumId;
  final String title;
  final int trackNumber;
  final int discNumber;
  final int durationMs;
  final String format;
  final int bitrate;
  final String? artistId;
  final String? artistName;
  final String? albumTitle;

  Duration get duration => Duration(milliseconds: durationMs);

  factory Track.fromJson(Map<String, dynamic> json) => Track(
        id: json['id']?.toString() ?? '',
        albumId: json['album_id']?.toString() ?? '',
        title: json['title']?.toString() ?? 'Untitled track',
        trackNumber: (json['track_number'] as num?)?.toInt() ?? 0,
        discNumber: (json['disc_number'] as num?)?.toInt() ?? 0,
        durationMs: (json['duration_ms'] as num?)?.toInt() ?? 0,
        format: json['format']?.toString() ?? '',
        bitrate: (json['bitrate'] as num?)?.toInt() ?? 0,
        artistId: json['artist_id']?.toString(),
        artistName: json['artist_name']?.toString(),
        albumTitle: json['album_title']?.toString(),
      );
}

class Playlist {
  const Playlist({
    required this.id,
    required this.name,
    this.userId = '',
    this.createdAt = '',
  });
  final String id;
  final String name;
  final String userId;
  final String createdAt;

  factory Playlist.fromJson(Map<String, dynamic> json) => Playlist(
        id: json['id']?.toString() ?? '',
        name: json['name']?.toString() ?? 'Playlist',
        userId: json['user_id']?.toString() ?? '',
        createdAt: json['created_at']?.toString() ?? '',
      );
}

class AuthResult {
  const AuthResult({required this.token, required this.user});
  final String token;
  final User user;

  factory AuthResult.fromJson(Map<String, dynamic> json) => AuthResult(
        token: json['token']?.toString() ?? '',
        user: User.fromJson((json['user'] as Map?)?.cast<String, dynamic>() ?? {}),
      );
}

class DashboardData {
  const DashboardData({
    required this.recentAlbums,
    required this.recentTracks,
    required this.favoriteTracks,
  });
  final List<Album> recentAlbums;
  final List<Track> recentTracks;
  final List<Track> favoriteTracks;

  factory DashboardData.fromJson(Map<String, dynamic> json) => DashboardData(
        recentAlbums: _list(json['recently_added_albums'], Album.fromJson),
        recentTracks: _list(json['recently_played_tracks'], Track.fromJson),
        favoriteTracks: _list(json['favorite_tracks'], Track.fromJson),
      );
}

class SearchResults {
  const SearchResults({
    required this.artists,
    required this.albums,
    required this.tracks,
  });
  final List<Artist> artists;
  final List<Album> albums;
  final List<Track> tracks;

  factory SearchResults.fromJson(Map<String, dynamic> json) => SearchResults(
        artists: _list(json['artists'], Artist.fromJson),
        albums: _list(json['albums'], Album.fromJson),
        tracks: _list(json['tracks'], Track.fromJson),
      );
}

class HeartDetails {
  const HeartDetails({
    required this.tracks,
    required this.albums,
    required this.artists,
    required this.playlists,
    required this.radio,
    required this.podcasts,
  });
  final List<Track> tracks;
  final List<Album> albums;
  final List<Artist> artists;
  final List<Playlist> playlists;
  final List<Map<String, dynamic>> radio;
  final List<Map<String, dynamic>> podcasts;

  factory HeartDetails.fromJson(Map<String, dynamic> json) => HeartDetails(
        tracks: _list(json['tracks'], Track.fromJson),
        albums: _list(json['albums'], Album.fromJson),
        artists: _list(json['artists'], Artist.fromJson),
        playlists: _list(json['playlists'], Playlist.fromJson),
        radio: _maps(json['radio']),
        podcasts: _maps(json['podcasts']),
      );
}

List<T> _list<T>(dynamic raw, T Function(Map<String, dynamic>) convert) {
  if (raw is! List) return <T>[];
  return raw
      .whereType<Map>()
      .map((item) => convert(item.cast<String, dynamic>()))
      .toList();
}

List<Map<String, dynamic>> _maps(dynamic raw) {
  if (raw is! List) return <Map<String, dynamic>>[];
  return raw.whereType<Map>().map((item) => item.cast<String, dynamic>()).toList();
}
