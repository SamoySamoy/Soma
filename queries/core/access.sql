-- name: GetMemberRole :one
SELECT role FROM space_members WHERE space_id = @space_id AND user_id = @user_id;
