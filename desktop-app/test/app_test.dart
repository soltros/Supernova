import 'package:flutter_test/flutter_test.dart';
import 'package:supernova_desktop/api_client.dart';

void main() {
  group('normalizeInstanceUrl', () {
    test('adds https to bare hosts', () {
      expect(normalizeInstanceUrl('music.example.com'), 'https://music.example.com');
    });

    test('preserves explicit local http URLs', () {
      expect(normalizeInstanceUrl('http://localhost:8080/'), 'http://localhost:8080');
    });

    test('rejects unsupported schemes', () {
      expect(() => normalizeInstanceUrl('ftp://music.example.com'), throwsFormatException);
    });
  });
}
