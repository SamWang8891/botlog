-- Fix: the original TTL overflowed DateTime's UInt32 range (max ~2106),
-- causing rows to be silently deleted during merges.
ALTER TABLE botlog.hits REMOVE TTL;
