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

- **Tweet Creation** - Create a new tweet with text content (max 255 characters)
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
6. Tweet content must not exceed 255 characters
7. Tweets are ordered by creation time (newest first) by default

### Non-Functional Requirements

1. Target response time: <100ms for single-record operations
2. Support concurrent connections via connection pooling
3. Horizontal scalability through stateless design

## Pagination

### Common Pagination Rules

All paginated endpoints follow these consistent rules:

1. **Cursor Default Value**: `-1`
   - When no `cursor` parameter is provided, the default value is `-1`
   - This represents the starting position (before the first item)

2. **Count Parameter**:
   - Default: `20` items per page
   - Maximum: `100` items per page
   - Minimum: `1` item per page

3. **Response Format**:
   - `count`: Number of items returned in the current page
   - `cursor`: Cursor value used in the current request
   - `next_cursor`: Cursor for the next page (null if no more pages)

4. **Next Page Detection**:
   - Fetch `count + 1` items from the database
   - If `count + 1` items are returned, a next page exists
   - Return only the first `count` items to the client
   - Calculate `next_cursor` as `cursor + count`

### Endpoints Using Cursor-Based Pagination

- `GET /tweets` - List all tweets
- `GET /users/me/feed` - Get authenticated user's feed

## Data Model

See [schema.md](./schema.md) for database schema details.
