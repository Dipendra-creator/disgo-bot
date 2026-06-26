package shared

import "github.com/bwmarrin/discordgo"

// This is the seed of the permission engine. Phase 1 ships Discord-native
// permission checks; later phases layer RBAC (custom roles, per-command
// overrides) on top of this same surface.

// MemberPermissions computes the aggregate Discord permission bitset for a
// member by OR-ing the @everyone role with each of the member's roles. The guild
// owner implicitly has all permissions.
func MemberPermissions(guild *discordgo.Guild, member *discordgo.Member) int64 {
	if guild == nil || member == nil {
		return 0
	}
	if member.User != nil && member.User.ID == guild.OwnerID {
		return discordgo.PermissionAll
	}

	perms := rolePermissions(guild, guild.ID) // @everyone shares the guild ID
	for _, rid := range member.Roles {
		perms |= rolePermissions(guild, rid)
	}
	if perms&discordgo.PermissionAdministrator != 0 {
		return discordgo.PermissionAll
	}
	return perms
}

// HasPermission reports whether the member holds the given permission in guild.
func HasPermission(guild *discordgo.Guild, member *discordgo.Member, perm int64) bool {
	return MemberPermissions(guild, member)&perm == perm
}

// RequirePermission returns a UserError when the member lacks the permission.
func RequirePermission(guild *discordgo.Guild, member *discordgo.Member, perm int64) error {
	if HasPermission(guild, member, perm) {
		return nil
	}
	return UserErr("You don't have permission to use this.")
}

func rolePermissions(guild *discordgo.Guild, roleID string) int64 {
	for _, r := range guild.Roles {
		if r.ID == roleID {
			return r.Permissions
		}
	}
	return 0
}
