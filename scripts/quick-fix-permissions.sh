#!/bin/bash

# SoulHound Quick Permission Fix Script
# Addresses the most common permission issues

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() { echo -e "${GREEN}[✓]${NC} $1"; }
print_error() { echo -e "${RED}[✗]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[!]${NC} $1"; }
print_info() { echo -e "${BLUE}[i]${NC} $1"; }

clear
echo "🚀 SoulHound Quick Permission Fix"
echo "================================="
echo ""

# Check if bot is running
print_info "Checking bot status..."
if docker ps | grep -q soulhound 2>/dev/null || podman ps | grep -q soulhound 2>/dev/null; then
    print_status "Bot is running"
else
    print_error "Bot is not running - start it first:"
    echo "   docker compose up -d"
    echo "   # or"
    echo "   ./scripts/deploy.sh"
    exit 1
fi

echo ""
echo "🔧 COMMON PERMISSION FIXES"
echo "=========================="
echo ""

# Fix 1: Zero Permissions Issue
echo "1️⃣  ZERO PERMISSIONS FIX"
echo "   If !debug shows 'Total calculated permissions: 0'"
echo ""
print_warning "IMMEDIATE ACTION REQUIRED:"
echo "   → Go to your Discord server"
echo "   → Server Settings → Roles"
echo "   → Find your bot's role (or create one)"
echo "   → Enable these permissions:"
echo "     ✅ View Channels"
echo "     ✅ Send Messages"
echo "     ✅ Connect"
echo "     ✅ Speak"
echo "     ✅ Read Message History"
echo "   → Assign the role to your bot"
echo ""

# Fix 2: Bot Re-invite
echo "2️⃣  RE-INVITE BOT WITH PERMISSIONS"
echo "   Use this URL to re-invite your bot:"
echo ""
print_info "🔗 https://discord.com/api/oauth2/authorize?client_id=YOUR_BOT_ID&permissions=3148800&scope=bot"
echo ""
print_warning "Replace YOUR_BOT_ID with your actual bot's Application ID"
echo "   Find it in Discord Developer Portal → Your App → General Information"
echo ""

# Fix 3: Developer Portal Settings
echo "3️⃣  DISCORD DEVELOPER PORTAL FIX"
echo "   Go to: https://discord.com/developers/applications"
echo "   → Select your bot application"
echo "   → Bot section → Enable 'MESSAGE CONTENT INTENT'"
echo "   → OAuth2 section → Select 'bot' scope + required permissions"
echo ""

# Fix 4: Channel Overrides
echo "4️⃣  CHANNEL PERMISSION OVERRIDES"
echo "   If bot works in some channels but not others:"
echo "   → Right-click problematic channel → Edit Channel"
echo "   → Permissions tab → Add your bot's role"
echo "   → Allow: View Channel, Send Messages, Connect, Speak"
echo ""

# Testing section
echo "🧪 TESTING YOUR FIXES"
echo "===================="
echo ""
echo "After making changes, test with:"
echo "   !debug    # Check permissions and voice states"
echo "   !help     # Verify bot can respond"
echo "   !play test # Test voice (must be in voice channel)"
echo ""

# Restart recommendation
print_warning "RESTART REQUIRED after permission changes:"
echo "   docker compose restart"
echo "   # or"
echo "   ./scripts/deploy.sh --restart"
echo ""

# Final tips
echo "💡 QUICK TIPS"
echo "============"
echo "• Wait 2-3 seconds after joining voice channel before using commands"
echo "• If bot has 0 permissions, it likely has no roles assigned"
echo "• Channel overrides can block server-wide permissions"
echo "• MESSAGE CONTENT INTENT is required in Developer Portal"
echo "• Re-inviting the bot often fixes permission issues"
echo ""

print_status "Run this script again after making changes to verify fixes!"
echo ""
echo "For detailed troubleshooting: ./scripts/troubleshoot-permissions.sh"
echo "For full documentation: docs/TROUBLESHOOTING.md"
