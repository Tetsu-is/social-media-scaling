# Social Media Scaling API Specification

## Overview

A high-performance REST API designed to handle social media operations at scale. Built with Go and PostgreSQL for efficient concurrent request handling.

## Features to Implement

### User Management

- **User Registration** - Create new user accounts with username and password
- **User Retrieval** - Fetch individual users by ID
- **User Listing** - Paginated list of all users

See [openapi.yaml](../openapi.yaml) for API endpoint details.

### Tweet Management

- **Tweet Creation** - Create a new tweet with text content (max 280 characters)
- **Tweet Listing** - Paginated list of tweets (supports filtering by user)

### Future Features

- Authentication (JWT-based)
- Follow/unfollow relationships
- Feed generation
- Rate limiting

## Requirements

### Functional Requirements

1. Users must have unique usernames
2. User IDs are auto-generated integers
3. All timestamps use ISO 8601 format
4. API responses use JSON (except health check)
5. Tweets must be associated with a valid user
6. Tweet content must not exceed 280 characters
7. Tweets are ordered by creation time (newest first) by default

### Non-Functional Requirements

1. Target response time: <100ms for single-record operations
2. Support concurrent connections via connection pooling
3. Horizontal scalability through stateless design

## Data Model

See [schema.md](./schema.md) for database schema details.
