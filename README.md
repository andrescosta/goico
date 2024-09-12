# Goico: The Jobico Framework
[![Go Report Card](https://goreportcard.com/badge/github.com/andrescosta/goico)](https://goreportcard.com/report/github.com/andrescosta/goico)
## Overview:

**Goico** is a framework designed to support the development of the Jobico family of services. It provides features for creating services, exposing REST or gRPC APIs, and building headless worker services.

## Key Features:

### 1. Service Creation and API Exposure:

Goico simplifies the development of microservices within the Jobico family of services. Developers can create services that expose REST or gRPC APIs, supporting a modular and scalable architecture.

### 2. WASM Runtime Based on WAZERO:

A core strength of Goico lies in its WebAssembly (WASM) runtime, built on the robust foundation of WAZERO. This runtime facilitates the execution of custom logic written in any WASM-supported programming language. 

### 3. Key/Value Embedded Database:

Goico integrates an embedded database based on [Pebble DB](https://github.com/cockroachdb/pebble), offering a key/value store for efficient data management. This embedded database serves as the backbone for storing critical information, supporting the reliable and fast retrieval of data essential for the operation of the Jobico products.

### 4. Streaming Capabilities for Database Updates:

Goico extends basic database functionality with advanced streaming for data updates. This allows real-time monitoring and reactions to changes in the embedded database, improving responsiveness and enabling dynamic adjustments within the Jobico ecosystem.

## Use Cases:

- **Microservices Development:**
  - Goico simplifies microservice creation, allowing developers to build modular components that enhance the functionality of the Jobico ecosystem.

- **Language-Agnostic WASM Execution:**
  - Using the WAZERO-based WASM runtime, Goico enables developers to implement Jobicolets in any language that compiles to WebAssembly, offering flexibility in event processing.

- **Efficient Data Storage and Retrieval:**
  - The embedded Pebble-based database provides a reliable key/value store for storing and retrieving essential data within Jobico.

- **Real-time Monitoring and Adaptation:**
  - Goico's database streaming features enable real-time reactions to changes, allowing Jobico to dynamically adapt to evolving requirements.
  