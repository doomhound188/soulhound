# SoulHound Troubleshooting Guide

## Voice Channel Detection Issues

If you're experiencing issues with the bot not detecting when you're in a voice channel, follow this troubleshooting guide.

### Common Error Messages

- "Error: You must be in a voice channel to use this command"
- "you must be in a voice channel to play music"
- "no guild available to join voice channel"

## Quick Fixes

### 1. Check Your Voice Channel Status
- **Make sure you're actually in a voice channel**
- **Try leaving and rejoining the voice channel**
- **Switch to a different voice channel and try again**

### 2. Use the Debug Command
```
!debug
```
This will show you:
- Current voice states in the server
- Bot permissions
- Guild information
- Voice connections

### 3. Check Bot Permissions

The bot needs these **essential Discord permissions**:

#### Required Permissions:
- ✅ **Read Messages** - To see your commands
- ✅ **Send Messages** - To respond to commands
- ✅ **Connect** - To join voice channels
- ✅ **Speak** - To play music
- ✅ **View Channels** - To see voice channels
- ✅ **Use Voice Activity** - For voice detection

#### Required Intents:
- ✅ **Guild Messages** - To read commands
- ✅ **Guild Voice States** - To detect voice channel membership
- ✅ **Message Content** - To read command content

## Step-by-Step Debugging

### Step 1: Verify Bot Invite URL

Make sure your bot was invited with the correct permissions URL:
```
https://discord.com/api/oauth2/authorize?client_id=YOUR_BOT_ID&permissions=3148800&scope=bot
```

Replace `YOUR_BOT_ID` with your actual bot's Application ID from the Discord Developer Portal.

## Permission Issues (Zero Permissions Error)

If the `!debug` command shows **"Total calculated permissions: 0"**, this means your bot has no effective permissions in the server. This is the most common cause of bot failures.

### Immediate Solutions

#### Option 1: Re-invite the Bot (Recommended)
1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Select your application
3. Go to the "OAuth2" section
4. Select "bot" scope
5. Select these permissions:
   - View Channels
   - Send Messages  
   - Connect
   - Speak
   - Read Message History
6. Copy the generated URL and visit it
7. Re-invite the bot to your server

#### Option 2: Fix Role Permissions
1. Go to your Discord server
2. Right-click the server name → "Server Settings"
3. Click "Roles" in the left sidebar
4. Find your bot's role (usually named after your bot)
5. Edit the role and enable required permissions:
   - View Channels
   - Send Messages
   - Connect
   - Speak
   - Read Message History
6. Save changes

#### Option 3: Create New Role for Bot
1. Server Settings → Roles → Create Role
2. Name it "Music Bot" or similar
3. Enable required permissions (see above)
4. Go to Members tab
5. Find your bot and assign the new role

### Discord Developer Portal Setup

#### Required Bot Settings:
1. **Bot Section:**
   - ✅ Bot enabled
   - ✅ MESSAGE CONTENT INTENT enabled
   - ✅ SERVER MEMBERS INTENT enabled (recommended)
   - ✅ PRESENCE INTENT enabled (recommended)

2. **OAuth2 Section:**
   - ✅ `bot` scope selected
   - ✅ Required permissions selected

### Channel-Specific Permission Issues

Sometimes the bot has server-wide permissions but is blocked by channel-specific overrides:

#### For Text Channels:
1. Right-click the text channel → "Edit Channel"
2. Go to "Permissions" tab
3. Add your bot's role or the bot directly
4. Set these to "Allow":
   - View Channel
   - Send Messages
   - Read Message History

#### For Voice Channels:
1. Right-click the voice channel → "Edit Channel"  
2. Go to "Permissions" tab
3. Add your bot's role or the bot directly
4. Set these to "Allow":
   - View Channel
   - Connect
   - Speak

### Advanced Permission Troubleshooting

#### Check Permission Hierarchy:
- Discord uses a permission hierarchy system
- Higher roles override lower roles
- Channel overrides beat role permissions
- Administrator permission bypasses all restrictions

#### Common Permission Values:
- **Basic Bot:** 2048 (Send Messages)
- **Music Bot:** 3148800 (Complete music bot permissions)
- **Admin Bot:** 8 (Administrator - not recommended for security)

#### Using Permission Calculator:
Visit https://discordapi.com/permissions.html to calculate custom permission values.

### Voice Channel Detection Issues

#### Symptoms:
- Bot says "You must be in a voice channel" when you are in one
- `!debug` shows "Voice States Count: 0"
- Bot can't see users in voice channels

#### Solutions:
1. **Wait After Joining:** Wait 2-3 seconds after joining a voice channel before using commands
2. **Refresh Voice State:** Leave and rejoin the voice channel
3. **Check Voice Channel Permissions:** Ensure bot can see the specific voice channel
4. **Restart Bot:** Sometimes Discord's voice state cache needs refreshing

### Testing Your Setup

After making permission changes, test with these commands:
```
!debug          # Check all permissions and voice states
!help           # Verify bot can send messages
!play test      # Test voice functionality (must be in voice channel)
```

### When to Restart the Bot

Restart the bot after:
- Changing bot permissions
- Modifying Discord Developer Portal settings
- Re-inviting the bot
- Major Discord server setting changes

```bash
# Using Docker Compose
docker compose restart

# Using deployment script
./scripts/deploy.sh --restart
```

### Getting Help

If you're still having issues after following this guide:

1. **Check Bot Logs:**
   ```bash
   docker logs soulhound-bot
   # or
   docker compose logs -f
   ```

2. **Run the Troubleshooting Script:**
   ```bash
   ./scripts/troubleshoot-permissions.sh
   ```

3. **Verify Discord Status:**
   Visit https://discordstatus.com to check for Discord API issues

4. **Common Issues Checklist:**
   - ☐ MESSAGE CONTENT INTENT enabled in Developer Portal
   - ☐ Bot re-invited with proper permissions
   - ☐ Server-level role permissions configured
   - ☐ Channel-specific overrides checked
   - ☐ Bot is in the same server as you
   - ☐ Waited after permission changes
   - ☐ Bot container is running
   - ☐ No Discord API outages

### Step 1: Verify Bot Invite URL (Detailed)

Make sure your bot was invited with the correct permissions. Use this invite URL format:

```
https://discord.com/api/oauth2/authorize?client_id=YOUR_BOT_ID&permissions=3148800&scope=bot
```

**Permission value `3148800` includes:**
- Read Messages (1024)
- Send Messages (2048)
- Connect (1048576)
- Speak (2097152)
- View Channels (1024)

### Step 2: Check Server Settings

1. **Go to Server Settings → Roles**
2. **Find your bot's role**
3. **Ensure it has these permissions:**
   - View Channels ✅
   - Send Messages ✅
   - Connect ✅
   - Speak ✅

### Step 3: Test with Debug Command

Run `!debug` and check the output:

```
!debug
```

**Expected output should show:**
- Your guild information
- List of users in voice channels (including you)
- Bot intents information

### Step 4: Voice Channel Permissions

1. **Right-click on the voice channel you're in**
2. **Select "Edit Channel" → Permissions**
3. **Make sure the bot role has:**
   - View Channel ✅
   - Connect ✅
   - Speak ✅

## Advanced Troubleshooting

### Issue: Bot Can't See Voice States

**Symptoms:**
- `!debug` shows "Voice States Count: 0" even when people are in voice channels
- Bot consistently says you're not in a voice channel

**Solutions:**
1. **Re-invite the bot** with correct permissions
2. **Check if the bot has the Guild Voice States intent**
3. **Restart the bot** to refresh the connection

### Issue: Bot Joins Wrong Voice Channel

**Symptoms:**
- Bot joins a voice channel but not the one you're in
- Audio doesn't play in your channel

**Solutions:**
1. **Leave and rejoin your voice channel**
2. **Use the bot in a different voice channel**
3. **Check if there are multiple voice channels with similar names**

### Issue: Intermittent Detection Problems

**Symptoms:**
- Sometimes works, sometimes doesn't
- Bot detects voice channel after a delay

**Solutions:**
1. **Wait a few seconds** after joining voice channel before using commands
2. **Try the command again** - there might be a caching delay
3. **Check Discord's server status** - API issues can cause delays

## Environment-Specific Issues

### Running in Docker/Containers

If you're running the bot in a container, ensure:
- Environment variables are properly set
- Bot token is correctly configured
- Network connectivity is working

### Self-Hosted vs Cloud

**Self-hosted bots** might have additional issues:
- Firewall blocking Discord API requests
- Network latency affecting voice state updates
- Insufficient system resources

## Getting Help

### Before Asking for Help

1. **Run `!debug` and share the output**
2. **Confirm you're in a voice channel**
3. **Check bot permissions using the steps above**
4. **Try with a different voice channel**

### Information to Include

When reporting issues, please include:
- Discord server ID
- Bot version/commit hash
- Error messages (exact text)
- Output from `!debug` command
- Steps you've already tried

### Common Solutions Summary

| Issue | Solution |
|-------|----------|
| "Must be in voice channel" | Check permissions, rejoin channel |
| Bot can't see voice states | Re-invite bot with correct permissions |
| Intermittent issues | Wait after joining, try again |
| Bot joins wrong channel | Leave/rejoin your voice channel |
| Debug shows no voice states | Check Guild Voice States intent |

## Bot Invite Checklist

When inviting the bot to your server:

- [ ] Bot has correct permissions (3148800)
- [ ] Bot role is above music-related roles
- [ ] Bot can see the channels you want to use
- [ ] Voice channel permissions are correctly set
- [ ] Bot has been granted Voice States intent

## Still Having Issues?

If you've tried all the above steps and are still having problems:

1. **Check Discord's status page** - Sometimes it's a Discord API issue
2. **Try in a different server** - This helps isolate server-specific issues
3. **Restart the bot** - Fresh connection might resolve state issues
4. **Check bot logs** - Look for error messages in the console

For persistent issues, consider:
- Creating a test server with minimal setup
- Checking if other voice bots work in your server
- Reviewing Discord's bot documentation for any recent changes

Remember: Most voice channel detection issues are permission-related. Double-check the bot's permissions in both the server settings and the specific voice channel you're trying to use.