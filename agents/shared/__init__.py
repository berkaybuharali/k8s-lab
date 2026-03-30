"""Shared package for Magic Cake agents."""
from .config import config, Config
from .redis_client import get_redis, RedisClient

__all__ = ["config", "Config", "get_redis", "RedisClient"]
