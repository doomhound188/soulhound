#!/bin/bash

# SoulHound Discord Bot Permission Troubleshooting Script
# This script helps diagnose and fix common permission issues

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[i]${NC} $1"
}

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

echo "🔧 SoulHound Discord Bot Permission Troubleshooting"
echo "=================================================="

# Check if bot is running
print_header "1. Bot Status Check"
if docker ps | grep -q soulhound 2>/dev/null; then
    print_status "Bot container is running"
elif podman ps | grep -q soulhound 2>/dev/null; then
    print_status "Bot container is running (Podman)"
else
    print_error "Bot container is not running"
    echo "   Run: docker compose up -d  or  ./scripts/deploy.sh"
    exit 1
fi

# Check Discord Developer Portal settings
print_header "2. Discord Developer Portal Checklist"
echo "Please verify the following in your Discord Developer Portal:"
echo "   📋 https://discord.com/developers/applications"
echo ""
echo "   Bot Section:"
echo "   ✅ Bot is enabled"
echo "   ✅ MESSAGE CONTENT INTENT is enabled"
echo "   ✅ SERVER MEMBERS INTENT is enabled (optional but recommended)"
echo "   ✅ PRESENCE INTENT is enabled (optional but recommended)"
echo ""
echo "   OAuth2 Section:"
echo "   ✅ bot scope is selected"
echo "   ✅ Required permissions are selected:"
echo "      - View Channels"
echo "      - Send Messages"
echo "      - Connect (Voice)"
echo "      - Speak (Voice)"
echo "      - Read Message History"
echo ""

# Generate proper invite URL
print_header "3. Bot Invite URL Generator"
echo "If your bot has permission issues, re-invite it with this URL:"
echo ""
echo "🔗 https://discord.com/api/oauth2/authorize?client_id=YOUR_BOT_ID&permissions=3148800&scope=bot"
echo ""
print_warning "Replace YOUR_BOT_ID with your actual bot's Application ID"
print_info "Permission value 3148800 includes: View Channels, Send Messages, Connect, Speak, Read Message History"

# Common permission issues and solutions
print_header "4. Common Permission Issues & Solutions"
echo ""
echo "❌ Issue: Bot shows 'Total calculated permissions: 0'"
echo "   🔧 Solution: The bot has no roles or roles have no permissions"
echo "   📋 Steps:"
echo "      1. Go to your Discord server"
echo "      2. Server Settings → Roles"
echo "      3. Find your bot's role (usually named after your bot)"
echo "      4. Edit the role and enable required permissions"
echo "      5. Or create a new role with permissions and assign it to the bot"
echo ""
echo "❌ Issue: Bot can't see voice channels"
echo "   🔧 Solution: Channel-specific permission overrides"
echo "   📋 Steps:"
echo "      1. Right-click on your voice channel → Edit Channel"
echo "      2. Go to Permissions tab"
echo "      3. Add your bot's role with Allow for:"
echo "         - View Channel"
echo "         - Connect"
echo "         - Speak"
echo ""
echo "❌ Issue: Bot can't send messages"
echo "   🔧 Solution: Text channel permissions"
echo "   📋 Steps:"
echo "      1. Right-click on your text channel → Edit Channel"
echo "      2. Go to Permissions tab"
echo "      3. Add your bot's role with Allow for:"
echo "         - View Channel"
echo "         - Send Messages"
echo "         - Read Message History"
echo ""

# Testing commands
print_header "5. Testing Commands"
echo "Once you've fixed permissions, test with these commands:"
echo ""
echo "   !debug        # Shows detailed permission and voice state info"
echo "   !help         # Shows all available commands"
echo "   !play test    # Try playing a song (must be in voice channel)"
echo ""

# Advanced troubleshooting
print_header "6. Advanced Troubleshooting"
echo ""
echo "🔍 Check bot logs:"
echo "   docker logs soulhound-bot"
echo "   # or"
echo "   docker compose logs -f"
echo ""
echo "🔍 Check container status:"
echo "   docker ps -a | grep soulhound"
echo ""
echo "🔍 Restart the bot:"
echo "   docker compose restart"
echo "   # or"
echo "   ./scripts/deploy.sh --restart"
echo ""
echo "🔍 Check Discord API status:"
echo "   https://discordstatus.com"
echo ""

# Permission bit calculator
print_header "7. Permission Bit Calculator"
echo "Common permission combinations:"
echo "   Basic Bot: 2048 (Send Messages)"
echo "   Music Bot: 3148800 (View Channels + Send Messages + Connect + Speak + Read Message History)"
echo "   Admin Bot: 8 (Administrator - not recommended)"
echo ""
echo "Calculate custom permissions:"
echo "   🔗 https://discordapi.com/permissions.html"
echo ""

# Final checklist
print_header "8. Final Checklist"
echo "Before asking for help, ensure you've:"
echo "   ☐ Enabled MESSAGE CONTENT INTENT in Discord Developer Portal"
echo "   ☐ Re-invited the bot with proper permissions"
echo "   ☐ Checked server-level role permissions"
echo "   ☐ Checked channel-specific permission overrides"
echo "   ☐ Verified the bot is in the same server as you"
echo "   ☐ Tried the !debug command"
echo "   ☐ Checked bot logs for errors"
echo "   ☐ Waited a few minutes after permission changes"
echo ""

print_info "If issues persist, check the bot logs and Discord Developer Portal settings"
print_info "Join our Discord server for support: [Add your support server invite]"

echo ""
echo "🚀 Good luck with your SoulHound bot setup!"
