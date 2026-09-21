import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:media_kit/media_kit.dart' show Media, Player;

import 'api_client.dart';
import 'models.dart';

class PlayerController extends ChangeNotifier {
  PlayerController() {
    _subscriptions.add(player.stream.playing.listen((value) {
      playing = value;
      notifyListeners();
    }));
    _subscriptions.add(player.stream.position.listen((value) {
      position = value;
      notifyListeners();
    }));
    _subscriptions.add(player.stream.duration.listen((value) {
      duration = value;
      notifyListeners();
    }));
  }

  final Player player = Player();
  final List<StreamSubscription<dynamic>> _subscriptions = [];
  final List<Track> queue = [];

  Track? currentTrack;
  String? externalTitle;
  String? externalSubtitle;
  bool playing = false;
  Duration position = Duration.zero;
  Duration duration = Duration.zero;
  int queueIndex = -1;
  Timer? _scrobbleTimer;

  Future<void> playTrack(ApiClient api, Track track, {List<Track>? context}) async {
    if (context != null) {
      queue
        ..clear()
        ..addAll(context);
      queueIndex = queue.indexWhere((item) => item.id == track.id);
    } else {
      queue
        ..clear()
        ..add(track);
      queueIndex = 0;
    }
    currentTrack = track;
    externalTitle = null;
    externalSubtitle = null;
    position = Duration.zero;
    duration = track.duration;
    notifyListeners();

    final url = await api.streamUrl(track.id);
    await player.open(Media(url), play: true);
    _scheduleScrobble(api, track);
  }

  Future<void> playExternal(String url, String title, {String? subtitle}) async {
    if (url.isEmpty) return;
    queue.clear();
    queueIndex = -1;
    currentTrack = null;
    externalTitle = title;
    externalSubtitle = subtitle;
    position = Duration.zero;
    duration = Duration.zero;
    _scrobbleTimer?.cancel();
    notifyListeners();
    await player.open(Media(url), play: true);
  }

  void _scheduleScrobble(ApiClient api, Track track) {
    _scrobbleTimer?.cancel();
    final target = track.durationMs <= 0
        ? const Duration(seconds: 30)
        : Duration(milliseconds: (track.durationMs ~/ 2).clamp(30000, 240000));
    _scrobbleTimer = Timer(target, () async {
      if (currentTrack?.id == track.id && playing) {
        try {
          await api.scrobble(track.id);
        } catch (_) {}
      }
    });
  }

  Future<void> toggle() => player.playOrPause();
  Future<void> seek(Duration value) => player.seek(value);

  Future<void> next(ApiClient api) async {
    if (queueIndex + 1 >= queue.length) return;
    final nextIndex = queueIndex + 1;
    final snapshot = List<Track>.from(queue);
    await playTrack(api, snapshot[nextIndex], context: snapshot);
    queueIndex = nextIndex;
  }

  Future<void> previous(ApiClient api) async {
    if (queueIndex <= 0) return;
    final previousIndex = queueIndex - 1;
    final snapshot = List<Track>.from(queue);
    await playTrack(api, snapshot[previousIndex], context: snapshot);
    queueIndex = previousIndex;
  }

  String get title => currentTrack?.title ?? externalTitle ?? 'Nothing playing';
  String get subtitle =>
      currentTrack?.artistName ?? externalSubtitle ?? currentTrack?.albumTitle ?? '';

  @override
  void dispose() {
    _scrobbleTimer?.cancel();
    for (final subscription in _subscriptions) {
      subscription.cancel();
    }
    player.dispose();
    super.dispose();
  }
}
