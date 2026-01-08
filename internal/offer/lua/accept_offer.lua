-- Improved version that doesn't error on "already accepted"
-- KEYS[1] = offer_group:{riderID}
-- ARGV[1] = driverID

local status = redis.call("HGET", KEYS[1], "status")

-- Offer group doesn't exist or expired
if not status then 
	return {err = "OFFER_GROUP_NOT_FOUND"}
end

-- Someone already accepted
if status == "accepted" then 
	local winner = redis.call("HGET", KEYS[1], "winner")
	if winner == ARGV[1] then
		return 1  -- Idempotent: this driver already won
	else 
		return 0  -- Someone else won (not an error, just lost the race)
	end
end

-- Already resolved with some other status
if status ~= "pending" then 
	return 0
end

-- Still pending - this driver wins!
redis.call("HSET", KEYS[1], "status", "accepted", "winner", ARGV[1])

return 1

-- Returns:
-- 1 = this driver won
-- 0 = this driver lost (someone else won first)
-- error = offer group not found/expired
-- status     -> pending | accepted | expired
-- winner     -> driverID
-- created_at -> timestamp
