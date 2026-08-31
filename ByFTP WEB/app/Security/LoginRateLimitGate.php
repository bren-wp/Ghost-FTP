<?php
declare(strict_types=1);

namespace ByFTP\Security;

/**
 * Applies login rate limits in abuse-resistant order.
 *
 * The IP budget is consumed first. Once that source is blocked, no account-specific
 * limiter is touched, preventing an already-blocked client from exhausting arbitrary
 * account budgets. The account budget is consumed only for requests admitted by IP.
 */
final class LoginRateLimitGate
{
    public static function consume(
        RateLimiter $ipLimiter,
        string $ipKey,
        RateLimiter $accountLimiter,
        string $accountKey
    ): bool {
        if (!$ipLimiter->consume($ipKey)) {
            return false;
        }
        return $accountLimiter->consume($accountKey);
    }
}
