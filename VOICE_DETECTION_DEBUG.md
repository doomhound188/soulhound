# Voice Detection Debugging Guide - SoulHound Bot

## Issue Analysis

The bot has proper **Gateway Intents** configured (`GUILD_VOICE_STATES` is enabled) and **Discord permissions** are correct, but voice channel detection is still failing. This indicates the issue is likely in the **voice state event processing** or **timing synchronization**.

## New Debugging Tools Added

### 1. **Enhanced Voice State Event Logging** 🔊
**What it does:** Provides detailed logging of all voice state events
**How to use:** Watch the bot console logs when joining/leaving voice channels

**Log messages to look for:**
```
🔊 VOICE STATE UPDATE HANDLER CALLED - UserID: [ID], ChannelID: [ID], GuildID: [ID]
🔊 User [ID] JOINED voice channel [ID] in guild [ID]
🔊 User [ID] LEFT voice channel in guild [ID]
🔊 All tracked voice states after update:
```

**What this tells us:**
- ✅ If you see these messages: Voice state events are being received
- ❌ If you see NO messages: Voice state events are not being received (intent/permission issue)

### 2. **Real-Time Voice Monitor** - `!voicemonitor`
**What it does:** Shows live comparison between internal tracking and Discord API
**How to use:** Run `!voicemonitor` while in a voice channel

**What it shows:**
- Your current voice state in internal tracking
- All tracked users in the guild
- Discord API current state
- Comparison between tracking and API data

**Testing process:**
1. Join a voice channel
2. Run `!voicemonitor`
3. Watch bot logs for voice state events
4. Run `!voicemonitor` again to see if tracking updated

### 3. **Voice State Refresh** - `!refreshvoice`
**What it does:** Forces a complete refresh of voice state data from Discord API
**How to use:** Run `!refreshvoice` to clear cache and fetch fresh data

**What it does:**
- Clears all cached voice states for the guild
- Makes fresh API call to Discord
- Rebuilds internal voice state tracking
- Shows before/after comparison

### 4. **Comprehensive Diagnostics** - `!diagnose`
**What it does:** Provides complete analysis of guild, channels, and permissions
**How to use:** Run `!diagnose` for full system analysis

**What it analyzes:**
- Guild information and member counts
- Bot status and role assignments
- Your user status and permissions
- All voice channels and their permissions
- Current voice states with detailed breakdown
- Specific recommendations based on findings

### 5. **API Connectivity Test** - `!apitest`
**What it does:** Tests Discord API connectivity and performance
**How to use:** Run `!apitest` to check API health

**What it tests:**
- API response times
- Cache vs API data comparison
- Rate limiting detection
- Voice state API functionality
- Network connectivity issues

### 6. **Simple Voice Test** - `!voicetest`
**What it does:** Quick voice state detection test
**How to use:** Run `!voicetest` for immediate voice state check

**What it shows:**
- Internal tracking status
- Discord API status
- Cache status
- Simple pass/fail for each method

## Diagnostic Workflow

### **Step 1: Check Event Reception**
1. Join a voice channel
2. Watch bot console logs
3. Look for `🔊 VOICE STATE UPDATE HANDLER CALLED` messages

**If you see the messages:**
- ✅ Events are being received
- Issue is in event processing logic
- Continue to Step 2

**If you see NO messages:**
- ❌ Events are not being received
- This is a Gateway Intent or permission issue
- Check Discord Developer Portal settings

### **Step 2: Test Voice Detection**
1. Run `!voicemonitor` while in voice channel
2. Check if internal tracking shows you
3. Check if Discord API shows you

**If API shows you but internal tracking doesn't:**
- Events are received but not processed correctly
- Check event handler logic

**If neither shows you:**
- Discord API synchronization issue
- Try `!refreshvoice` to force refresh

### **Step 3: Comprehensive Analysis**
1. Run `!diagnose` for full system analysis
2. Check all voice channels and permissions
3. Verify bot can see the channels you're using

### **Step 4: API Health Check**
1. Run `!apitest` to check API connectivity
2. Look for slow response times or errors
3. Check for rate limiting issues

## Common Issues and Solutions

### **Issue: No voice state events received**
**Symptoms:** No `🔊` log messages when joining/leaving voice
**Cause:** Gateway Intents not properly configured
**Solution:** 
- Check Discord Developer Portal → Bot → Privileged Gateway Intents
- Ensure `GUILD_VOICE_STATES` intent is enabled in code
- Restart bot after changes

### **Issue: Events received but detection fails**
**Symptoms:** See `🔊` logs but `!voicemonitor` shows not tracked
**Cause:** Event processing logic error
**Solution:**
- Check event handler implementation
- Verify voice state storage logic
- Use `!refreshvoice` to reset state

### **Issue: API shows user but bot doesn't detect**
**Symptoms:** `!diagnose` shows you in API but commands fail
**Cause:** Cache synchronization issue
**Solution:**
- Use `!refreshvoice` to sync cache
- Check for timing issues
- Verify event handler is updating cache

### **Issue: Slow or failing API calls**
**Symptoms:** `!apitest` shows slow response times
**Cause:** Network or Discord API issues
**Solution:**
- Check network connectivity
- Verify Discord API status
- Implement retry logic

## Testing Commands Summary

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `!voicemonitor` | Real-time voice state monitoring | Primary debugging tool |
| `!voicetest` | Quick voice detection test | Fast status check |
| `!refreshvoice` | Force refresh voice data | When cache is out of sync |
| `!diagnose` | Complete system analysis | Comprehensive troubleshooting |
| `!apitest` | API connectivity test | When suspecting API issues |
| `!debug` | General bot debug info | Basic troubleshooting |

## Next Steps

1. **Test the new debugging tools** by running them while in a voice channel
2. **Watch the console logs** for voice state event messages
3. **Use the diagnostic workflow** to identify the specific issue
4. **Report findings** - the enhanced logging will help identify exactly where the voice detection is failing

The bot now has comprehensive voice detection debugging capabilities that should help identify and resolve the voice channel detection issue.
