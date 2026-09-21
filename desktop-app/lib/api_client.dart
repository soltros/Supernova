import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

import 'models.dart';

class ApiException implements Exception {
  ApiException(this.message, {this.statusCode});
  final String message;
  final int? statusCode;

  @override
  String toString() => message;
}

String normalizeInstanceUrl(String input) {
  var value = input.trim();
  if (value.isEmpty) return '';
  if (!value.contains('://')) value = 'https://$value';
  final uri = Uri.tryParse(value);
  if (uri == null || !uri.hasScheme || uri.host.isEmpty) {
    throw const FormatException('Enter a valid Supernova instance URL');
  }
  if (uri.scheme != 'http' && uri.scheme != 'https') {
    throw const FormatException('Instance URL must use http or https');
  }
  final path = uri.path == '/' ? '' : uri.path.replaceAll(RegExp(r'/+$'), '');
  return uri.replace(path: path, query: null, fragment: null).toString().replaceAll(RegExp(r'/+$'), '');
}

class ApiClient {
  ApiClient({required this.baseUrl, this.token});
  final String baseUrl;
  final String? token;

  Uri _uri(String path, [Map<String, String?> query = const {}]) {
    final filtered = <String, String>{};
    for (final entry in query.entries) {
      if (entry.value != null && entry.value!.isNotEmpty) filtered[entry.key] = entry.value!;
    }
    return Uri.parse('$baseUrl$path').replace(queryParameters: filtered.isEmpty ? null : filtered);
  }

  Future<http.Response> _send(
    String method,
    String path, {
    Map<String, String?> query = const {},
    Object? body,
    bool auth = true,
  }) async {
    final request = http.Request(method, _uri(path, query));
    request.headers.addAll({
      'Accept': 'application/json',
      if (body != null) 'Content-Type': 'application/json',
      if (auth && token?.isNotEmpty == true) 'Authorization': 'Bearer $token',
    });
    if (body != null) request.body = jsonEncode(body);
    try {
      final streamed = await request.send().timeout(const Duration(seconds: 20));
      final response = await http.Response.fromStream(streamed);
      if (response.statusCode < 200 || response.statusCode >= 300) {
        final message = response.body.trim().isEmpty
            ? 'Supernova returned HTTP ${response.statusCode}'
            : response.body.trim();
        throw ApiException(message, statusCode: response.statusCode);
      }
      return response;
    } on SocketException {
      throw ApiException('Could not reach $baseUrl');
    }
  }

  dynamic _json(http.Response response) =>
      response.body.isEmpty ? <String, dynamic>{} : jsonDecode(response.body);

  Future<void> health() async => _send('GET', '/api/health', auth: false);

  Future<AuthResult> login(String username, String password) async {
    final response = await _send(
      'POST',
      '/api/auth/login',
      auth: false,
      body: {'username': username, 'password': password},
    );
    return AuthResult.fromJson((_json(response) as Map).cast<String, dynamic>());
  }

  Future<AuthResult> register(String username, String password, String inviteCode) async {
    final response = await _send(
      'POST',
      '/api/auth/register',
      auth: false,
      body: {'username': username, 'password': password, 'invite_code': inviteCode},
    );
    return AuthResult.fromJson((_json(response) as Map).cast<String, dynamic>());
  }

  Future<User> me() async {
    final response = await _send('GET', '/api/auth/me');
    return User.fromJson((_json(response) as Map).cast<String, dynamic>());
  }

  Future<void> logout() async => _send('POST', '/api/auth/logout');

  Future<void> changePassword(String current, String next) async =>
      _send('POST', '/api/auth/change-password', body: {'current': current, 'new': next});

  Future<List<Album>> albums({int limit = 100, int offset = 0, String? artistId}) async {
    final response = await _send('GET', '/api/albums', query: {
      'limit': '$limit',
      'offset': '$offset',
      'artist_id': artistId,
    }, auth: false);
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => Album.fromJson(x.cast<String, dynamic>()))
        .toList();
  }

  Future<List<Album>> allAlbums({String? artistId}) async {
    final out = <Album>[];
    for (var offset = 0;; offset += 200) {
      final page = await albums(limit: 200, offset: offset, artistId: artistId);
      out.addAll(page);
      if (page.length < 200) return out;
    }
  }

  Future<List<Artist>> artists({int limit = 100, int offset = 0, String? letter}) async {
    final response = await _send('GET', '/api/artists', query: {
      'limit': '$limit',
      'offset': '$offset',
      'letter': letter,
    }, auth: false);
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => Artist.fromJson(x.cast<String, dynamic>()))
        .toList();
  }

  Future<List<Artist>> allArtists() async {
    final out = <Artist>[];
    for (var offset = 0;; offset += 200) {
      final page = await artists(limit: 200, offset: offset);
      out.addAll(page);
      if (page.length < 200) return out;
    }
  }

  Future<List<Track>> tracks({String? albumId, String? artistId, int limit = 200, int offset = 0}) async {
    final response = await _send('GET', '/api/tracks', query: {
      'album_id': albumId,
      'artist_id': artistId,
      'limit': '$limit',
      'offset': '$offset',
    }, auth: false);
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => Track.fromJson(x.cast<String, dynamic>()))
        .toList();
  }

  Future<List<Track>> allTracks({String? albumId, String? artistId}) async {
    final out = <Track>[];
    for (var offset = 0;; offset += 300) {
      final page = await tracks(albumId: albumId, artistId: artistId, limit: 300, offset: offset);
      out.addAll(page);
      if (page.length < 300) return out;
    }
  }

  Future<SearchResults> search(String query) async {
    final response = await _send('GET', '/api/search', query: {'q': query}, auth: false);
    return SearchResults.fromJson((_json(response) as Map).cast<String, dynamic>());
  }

  Future<DashboardData> dashboard() async {
    final response = await _send('GET', '/api/dashboard');
    return DashboardData.fromJson((_json(response) as Map).cast<String, dynamic>());
  }

  Future<HeartDetails> heartDetails() async {
    final response = await _send('GET', '/api/hearts/details');
    return HeartDetails.fromJson((_json(response) as Map).cast<String, dynamic>());
  }

  Future<void> heart(String type, String id, {Map<String, dynamic>? metadata}) async =>
      _send('POST', '/api/hearts', body: {
        'entity_type': type,
        'entity_id': id,
        'metadata': ?metadata,
      });

  Future<void> unheart(String type, String id) async =>
      _send('DELETE', '/api/hearts', query: {'entity_type': type, 'entity_id': id});

  Future<List<Playlist>> playlists() async {
    final response = await _send('GET', '/api/playlists');
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => Playlist.fromJson(x.cast<String, dynamic>()))
        .toList();
  }

  Future<Playlist> createPlaylist(String name) async {
    final response = await _send('POST', '/api/playlists', body: {'name': name});
    return Playlist.fromJson((_json(response) as Map).cast<String, dynamic>());
  }

  Future<void> deletePlaylist(String id) async => _send('DELETE', '/api/playlists/$id');

  Future<List<Track>> playlistTracks(String id) async {
    final response = await _send('GET', '/api/playlists/$id/tracks');
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => Track.fromJson(x.cast<String, dynamic>()))
        .toList();
  }

  Future<void> addTrackToPlaylist(String playlistId, String trackId) async =>
      _send('POST', '/api/playlists/$playlistId/tracks', body: {'track_id': trackId});

  Future<void> removeTrackFromPlaylist(String playlistId, String trackId) async =>
      _send('DELETE', '/api/playlists/$playlistId/tracks/$trackId');

  Future<String> mediaTicket(String scope, String resource) async {
    final response = await _send(
      'POST',
      '/api/media-ticket',
      body: {'scope': scope, 'resource': resource},
    );
    return ((_json(response) as Map)['ticket'] ?? '').toString();
  }

  Future<String> streamUrl(String trackId) async {
    final ticket = await mediaTicket('stream', trackId);
    return '$baseUrl/api/stream/$trackId?ticket=${Uri.encodeQueryComponent(ticket)}';
  }

  String albumArtUrl(String albumId) => '$baseUrl/api/art/album/$albumId';

  Future<void> scrobble(String trackId) async =>
      _send('POST', '/api/scrobbles', body: {'track_id': trackId});

  Future<List<Map<String, dynamic>>> plugins() async {
    final response = await _send('GET', '/api/plugins');
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => x.cast<String, dynamic>())
        .toList();
  }

  Future<void> scanLibrary() async => _send('POST', '/api/scan');

  Future<Map<String, dynamic>> scanProgress() async {
    final response = await _send('GET', '/api/scan/progress');
    return (_json(response) as Map).cast<String, dynamic>();
  }

  Future<void> resetArtists() async => _send('POST', '/api/settings/reset-artists');

  Future<Map<String, dynamic>> maintenancePreview(String pluginId) async {
    final response = await _send('GET', '/api/plugins/$pluginId/preview');
    return (_json(response) as Map).cast<String, dynamic>();
  }

  Future<List<Map<String, dynamic>>> podcastSubscriptions() async {
    final response = await _send('GET', '/api/plugins/podcasts/subscriptions');
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => x.cast<String, dynamic>())
        .toList();
  }

  Future<List<Map<String, dynamic>>> podcastSearch(String query) async {
    final response = await _send('GET', '/api/plugins/podcasts/search', query: {'q': query});
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => x.cast<String, dynamic>())
        .toList();
  }

  Future<List<Map<String, dynamic>>> podcastEpisodes(String feedId) async {
    final response = await _send('GET', '/api/plugins/podcasts/episodes', query: {'id': feedId});
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => x.cast<String, dynamic>())
        .toList();
  }

  Future<void> subscribePodcast(Map<String, dynamic> feed) async => _send(
        'POST',
        '/api/plugins/podcasts/subscriptions',
        body: {
          'feed_id': (feed['id'] ?? '').toString(),
          'feed_url': (feed['url'] ?? feed['feedUrl'] ?? '').toString(),
          'title': (feed['title'] ?? 'Podcast').toString(),
          'image_url': (feed['image'] ?? feed['artwork'] ?? '').toString(),
        },
      );

  Future<void> unsubscribePodcast(String feedId) async =>
      _send('DELETE', '/api/plugins/podcasts/subscriptions', query: {'feed_id': feedId});

  Future<List<Map<String, dynamic>>> radioSubscriptions() async {
    final response = await _send('GET', '/api/plugins/radio/subscriptions');
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => x.cast<String, dynamic>())
        .toList();
  }

  Future<List<Map<String, dynamic>>> radioSearch(String query) async {
    final response = await _send('GET', '/api/plugins/radio/search', query: {'q': query, 'limit': '50'});
    return (jsonDecode(response.body) as List)
        .whereType<Map>()
        .map((x) => x.cast<String, dynamic>())
        .toList();
  }

  Future<void> subscribeRadio(Map<String, dynamic> station) async => _send(
        'POST',
        '/api/plugins/radio/subscriptions',
        body: {
          'station_id': (station['stationuuid'] ?? station['station_id'] ?? '').toString(),
          'url': (station['url_resolved'] ?? station['url'] ?? '').toString(),
          'name': (station['name'] ?? 'Radio station').toString(),
          'favicon': (station['favicon'] ?? '').toString(),
        },
      );

  Future<void> unsubscribeRadio(String stationId) async =>
      _send('DELETE', '/api/plugins/radio/subscriptions', query: {'station_id': stationId});
}
