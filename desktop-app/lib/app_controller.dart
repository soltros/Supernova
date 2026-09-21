import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'api_client.dart';
import 'models.dart';

class AppController extends ChangeNotifier {
  AppController({
    FlutterSecureStorage? secureStorage,
    SharedPreferencesAsync? preferences,
  })  : _secureStorage = secureStorage ?? const FlutterSecureStorage(),
        _preferences = preferences ?? SharedPreferencesAsync();

  static const _instanceKey = 'supernova.instance';
  static const _tokenKey = 'supernova.token';

  final FlutterSecureStorage _secureStorage;
  final SharedPreferencesAsync _preferences;

  bool booting = true;
  bool busy = false;
  String? error;
  String instance = '';
  String? token;
  User? user;

  bool get authenticated => user != null && token != null && instance.isNotEmpty;

  ApiClient get api {
    if (instance.isEmpty) throw StateError('No Supernova instance configured');
    return ApiClient(baseUrl: instance, token: token);
  }

  Future<void> boot() async {
    try {
      instance = await _preferences.getString(_instanceKey) ?? '';
      token = await _secureStorage.read(key: _tokenKey);
      if (instance.isNotEmpty && token != null && token!.isNotEmpty) {
        try {
          user = await api.me();
        } catch (_) {
          await _secureStorage.delete(key: _tokenKey);
          token = null;
        }
      }
    } finally {
      booting = false;
      notifyListeners();
    }
  }

  Future<void> login({
    required String instanceUrl,
    required String username,
    required String password,
  }) async {
    await _authenticate(
      instanceUrl: instanceUrl,
      action: (client) => client.login(username.trim(), password),
    );
  }

  Future<void> register({
    required String instanceUrl,
    required String username,
    required String password,
    required String inviteCode,
  }) async {
    await _authenticate(
      instanceUrl: instanceUrl,
      action: (client) => client.register(username.trim(), password, inviteCode.trim()),
    );
  }

  Future<void> _authenticate({
    required String instanceUrl,
    required Future<AuthResult> Function(ApiClient client) action,
  }) async {
    busy = true;
    error = null;
    notifyListeners();
    try {
      final normalized = normalizeInstanceUrl(instanceUrl);
      final client = ApiClient(baseUrl: normalized);
      await client.health();
      final result = await action(client);
      instance = normalized;
      token = result.token;
      user = result.user;
      await _preferences.setString(_instanceKey, instance);
      await _secureStorage.write(key: _tokenKey, value: token);
    } catch (e) {
      error = e.toString();
      rethrow;
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  Future<void> logout() async {
    busy = true;
    notifyListeners();
    try {
      try {
        await api.logout();
      } catch (_) {}
      token = null;
      user = null;
      await _secureStorage.delete(key: _tokenKey);
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  Future<void> forgetInstance() async {
    await logout();
    instance = '';
    await _preferences.remove(_instanceKey);
    notifyListeners();
  }
}
