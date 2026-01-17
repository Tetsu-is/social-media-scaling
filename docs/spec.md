# Social Media Scaling API Specification

## Overview

A high-performance REST API designed to handle social media operations at scale. Built with Go and PostgreSQL for efficient concurrent request handling.

## Features to Implement

### User Management

- **User Registration** - Create new user accounts with username and password
- **User Retrieval** - Fetch individual users by ID
- **User Listing** - Paginated list of all users

See [openapi.yaml](../openapi.yaml) for API endpoint details.

### Future Features

- Authentication (JWT-based)
- Posts/content creation
- Follow/unfollow relationships
- Feed generation
- Rate limiting

## Requirements

### Functional Requirements

1. Users must have unique usernames
2. User IDs are auto-generated integers
3. All timestamps use ISO 8601 format
4. API responses use JSON (except health check)

### Non-Functional Requirements

1. Target response time: <100ms for single-record operations
2. Support concurrent connections via connection pooling
3. Horizontal scalability through stateless design

## Data Model

See [schema.md](./schema.md) for database schema details.
