# Audio Transcription - Manual Testing Checklist

This document provides a comprehensive manual testing checklist for the audio transcription feature.

## Prerequisites

Before testing, ensure:

- [ ] WhatsApp bot is running and connected
- [ ] Transcription service is running at `http://localhost:8009` (or configured URL)
- [ ] You have access to the bot owner WhatsApp account
- [ ] Database at `/data/auth.db` is accessible and writable

## Test Environment Setup

1. **Start Transcription Service:**
   ```bash
   # Ensure the transcription service is running
   curl http://localhost:8009/health  # Should return 200 OK
   ```

2. **Start WhatsApp Bot:**
   ```bash
   # With environment variable
   TRANSCRIBE_SERVICE_URL=http://localhost:8009 ./whatsapp-livetranslate

   # Or using .env file
   ./whatsapp-livetranslate
   ```

3. **Verify Bot Connection:**
   - [ ] Bot shows "Connected" status
   - [ ] QR code scanned successfully (if first run)
   - [ ] No connection errors in logs

## Phase 1: Command Testing

### Test 1.1: Enable Transcription (Default Language)
**Steps:**
1. Send `/tenable` command to bot (as owner)

**Expected Results:**
- [ ] Bot responds with success message: "✅ Audio transcription enabled for this chat (language: he)"
- [ ] No errors in bot logs

**Verification:**
```sql
SELECT * FROM transcription_settings WHERE chat_jid = 'your_chat_jid';
-- Should show: enabled=1, language='he'
```

### Test 1.2: Enable Transcription (Custom Language)
**Steps:**
1. Send `/tenable en` command to bot (as owner)

**Expected Results:**
- [ ] Bot responds with success message: "✅ Audio transcription enabled for this chat (language: en)"
- [ ] Database updated with language='en'

### Test 1.3: Disable Transcription
**Steps:**
1. Send `/tdisable` command to bot (as owner)

**Expected Results:**
- [ ] Bot responds with success message: "❌ Audio transcription disabled for this chat"
- [ ] Database shows enabled=0

### Test 1.4: Non-Owner Access Control
**Steps:**
1. Send `/tenable` command from non-owner account

**Expected Results:**
- [ ] Bot does not respond (silent fail) OR shows permission denied
- [ ] Transcription is NOT enabled in database

## Phase 2: Audio Transcription Testing

### Test 2.1: Basic Hebrew Audio Transcription
**Prerequisites:**
- [ ] Transcription enabled with `/tenable` (defaults to Hebrew)

**Steps:**
1. Send a Hebrew voice message to the bot

**Expected Results:**
- [ ] Bot replies with transcription in format:
  ```
  🎤 *Transcription:*

  [transcribed text in Hebrew]
  ```
- [ ] Transcription is reasonably accurate
- [ ] Response time < 10 seconds for short audio

**Logs to Check:**
- [ ] No errors about audio download
- [ ] No errors about transcription service
- [ ] Model used: `ivrit-ct2`
- [ ] Endpoint: `/api/transcribe/whisper-ivrit`

### Test 2.2: English Audio Transcription
**Prerequisites:**
- [ ] Transcription enabled with `/tenable en`

**Steps:**
1. Send an English voice message to the bot

**Expected Results:**
- [ ] Bot replies with English transcription
- [ ] Model used: `whisper-v3-turbo`
- [ ] Endpoint: `/api/transcribe/whisper`

### Test 2.3: Language Detection
**Prerequisites:**
- [ ] Transcription enabled with `/tenable en`

**Steps:**
1. Send a Hebrew voice message (when English is configured)

**Expected Results:**
- [ ] Bot transcribes the audio
- [ ] If detected language differs, shows: "🌐 Detected language: he"

### Test 2.4: Disabled Transcription
**Prerequisites:**
- [ ] Transcription disabled with `/tdisable`

**Steps:**
1. Send a voice message to the bot

**Expected Results:**
- [ ] Bot does NOT reply with transcription
- [ ] Voice message is NOT processed as a command
- [ ] Logs show no transcription attempt

### Test 2.5: PTT Voice Note vs Audio Message
**Steps:**
1. Send a PTT (push-to-talk) voice note
2. Send a regular audio file (e.g., forwarded audio)

**Expected Results:**
- [ ] Both types are transcribed (if enabled)
- [ ] Both trigger `shouldTranscribe()` check

## Phase 3: Error Handling Testing

### Test 3.1: Transcription Service Down
**Steps:**
1. Stop the transcription service
2. Enable transcription with `/tenable`
3. Send a voice message

**Expected Results:**
- [ ] Bot does NOT reply with transcription
- [ ] Error logged: "Transcription failed for chat [chat_id]: [error details]"
- [ ] Bot does NOT crash or show error to user (silent fail)
- [ ] User does not see error message

### Test 3.2: Invalid Audio Format
**Steps:**
1. Send a corrupted or invalid audio file

**Expected Results:**
- [ ] Bot handles gracefully (silent fail or error logged)
- [ ] No crash or unhandled exception

### Test 3.3: Very Long Audio (>5 minutes)
**Steps:**
1. Send a voice message longer than 5 minutes

**Expected Results:**
- [ ] Bot attempts transcription
- [ ] May timeout (5-minute timeout configured)
- [ ] If timeout, error is logged but user not notified

### Test 3.4: Network Issues
**Steps:**
1. Simulate network latency to transcription service
2. Send voice message

**Expected Results:**
- [ ] Bot waits up to 5 minutes
- [ ] If timeout, fails gracefully

## Phase 4: Multi-Chat Testing

### Test 4.1: Per-Chat Settings Isolation
**Steps:**
1. Enable transcription in Chat A with `/tenable he`
2. Enable transcription in Chat B with `/tenable en`
3. Send Hebrew voice in Chat A
4. Send English voice in Chat B

**Expected Results:**
- [ ] Chat A transcribes with Hebrew model (ivrit-ct2)
- [ ] Chat B transcribes with English model (whisper-v3-turbo)
- [ ] Settings do not interfere with each other

**Verification:**
```sql
SELECT chat_jid, enabled, language FROM transcription_settings;
-- Should show different settings per chat
```

### Test 4.2: Disable in One Chat
**Steps:**
1. Enable in Chat A and Chat B
2. Disable only Chat A with `/tdisable`
3. Send voice messages to both chats

**Expected Results:**
- [ ] Chat A: No transcription
- [ ] Chat B: Transcription works normally

## Phase 5: Integration Testing

### Test 5.1: Audio Not Processed as Command
**Steps:**
1. Enable transcription
2. Send a voice message that says "/help" or another command

**Expected Results:**
- [ ] Bot transcribes the audio
- [ ] Bot does NOT execute the command
- [ ] `handleMessage()` returns early after transcription

### Test 5.2: Transcription + Translation
**Steps:**
1. Enable transcription
2. Send Hebrew voice message
3. Reply to transcription with `/en` (translation command)

**Expected Results:**
- [ ] Voice message is transcribed
- [ ] Translation command works on the transcription text

### Test 5.3: Concurrent Requests
**Steps:**
1. Send 3-5 voice messages rapidly from different chats

**Expected Results:**
- [ ] All messages are transcribed
- [ ] No race conditions or database locks
- [ ] Responses may be delayed but all complete

## Phase 6: Database Testing

### Test 6.1: Database Persistence
**Steps:**
1. Enable transcription with `/tenable he`
2. Restart the bot
3. Send a voice message

**Expected Results:**
- [ ] Settings persist after restart
- [ ] Transcription still works with saved language

### Test 6.2: Database Integrity
**Steps:**
1. Enable/disable transcription multiple times
2. Check database

**Expected Results:**
- [ ] No duplicate entries per chat_jid
- [ ] UNIQUE constraint enforced on chat_jid
- [ ] Updates work correctly (no inserts on enable after already enabled)

**Verification:**
```sql
-- Should return only 1 row per chat
SELECT chat_jid, COUNT(*) as count
FROM transcription_settings
GROUP BY chat_jid
HAVING count > 1;
```

## Phase 7: Performance Testing

### Test 7.1: Response Time
**Steps:**
1. Send 5-second voice message
2. Measure time from send to transcription response

**Expected Results:**
- [ ] Response time: 5-15 seconds for 5-second audio
- [ ] No significant delays beyond transcription processing

### Test 7.2: Memory Usage
**Steps:**
1. Monitor bot memory before/after transcription
2. Send multiple voice messages

**Expected Results:**
- [ ] Temporary files cleaned up (check `/tmp/whatsapp_audio_*.ogg`)
- [ ] No memory leaks
- [ ] Memory returns to baseline after processing

### Test 7.3: Concurrent Transcriptions
**Steps:**
1. Send 3 voice messages simultaneously from different chats

**Expected Results:**
- [ ] All messages processed
- [ ] No blocking or deadlocks
- [ ] Responses may be sequential (due to service limitations)

## Phase 8: Edge Cases

### Test 8.1: Empty Voice Message
**Steps:**
1. Send a very short (< 1 second) voice note

**Expected Results:**
- [ ] Transcription attempted
- [ ] May return empty or "..." text
- [ ] No crash

### Test 8.2: Non-Voice Audio
**Steps:**
1. Send a music file or sound effect as audio message

**Expected Results:**
- [ ] Transcription attempted
- [ ] May return nonsensical text or empty
- [ ] No crash

### Test 8.3: Multiple Languages in Same Audio
**Steps:**
1. Send voice with mixed Hebrew and English

**Expected Results:**
- [ ] Transcription captures both languages (depends on model)
- [ ] Detected language may show primary language

### Test 8.4: Special Characters in Transcription
**Steps:**
1. Send voice with numbers, punctuation, special terms

**Expected Results:**
- [ ] Special characters handled correctly
- [ ] No encoding issues in WhatsApp message

## Regression Testing

After any code changes, verify:

- [ ] All unit tests pass: `go test ./internal/services/transcription/...`
- [ ] Integration tests pass: `go test ./test/integration/...`
- [ ] No new compiler warnings
- [ ] Bot builds successfully: `go build -o whatsapp-livetranslate .`

## Known Limitations

Document any known issues or limitations:

1. **Timeout:** 5-minute maximum for transcription
2. **Silent Failures:** Service errors not shown to users
3. **Model Selection:** Automatic based on language (cannot override)
4. **Concurrent Limit:** May be limited by transcription service capacity

## Test Results Template

Use this template to document test results:

```
Test Date: YYYY-MM-DD
Tester: [Name]
Bot Version: [Git Commit Hash]
Transcription Service Version: [Version]

Results:
- Phase 1: ✅ PASS (5/5 tests)
- Phase 2: ✅ PASS (5/5 tests)
- Phase 3: ✅ PASS (4/4 tests)
- Phase 4: ✅ PASS (2/2 tests)
- Phase 5: ✅ PASS (3/3 tests)
- Phase 6: ✅ PASS (2/2 tests)
- Phase 7: ✅ PASS (3/3 tests)
- Phase 8: ⚠️  PARTIAL (3/4 tests - empty voice issue)

Issues Found:
- [Issue #1]: Description
- [Issue #2]: Description

Notes:
- [Any additional observations]
```

## Automated Testing Commands

Quick commands for automated verification:

```bash
# Run all tests
IS_DOCKER=true GEMINI_API_KEY=test_key go test ./...

# Run only transcription tests
IS_DOCKER=true GEMINI_API_KEY=test_key go test ./internal/services/transcription/...

# Run integration tests
IS_DOCKER=true GEMINI_API_KEY=test_key go test ./test/integration/...

# Run with coverage
IS_DOCKER=true GEMINI_API_KEY=test_key go test -cover ./internal/services/transcription/...

# Check database schema
sqlite3 /data/auth.db "SELECT sql FROM sqlite_master WHERE name='transcription_settings';"
```

---

**Testing Status:** Phase 7 - Complete
**Last Updated:** 2025-01-18
