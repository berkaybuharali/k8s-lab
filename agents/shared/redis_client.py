"""Shared Redis client factory for Magic Cake agents."""
import redis
from .config import config


class RedisClient:
    """Redis connection factory."""

    _client = None

    @classmethod
    def get_client(cls) -> redis.Redis:
        """Get or create Redis client instance."""
        if cls._client is None:
            cls._client = redis.Redis(
                host=config.REDIS_HOST,
                port=config.REDIS_PORT,
                decode_responses=True,
                socket_connect_timeout=5,
                socket_timeout=5,
            )
        return cls._client

    @classmethod
    def ping(cls) -> bool:
        """Test Redis connection."""
        try:
            return cls.get_client().ping()
        except Exception:
            return False


def get_redis() -> redis.Redis:
    """Convenience function to get Redis client."""
    return RedisClient.get_client()
