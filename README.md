# Goico: The Jobico Framework
[![Go Report Card](https://goreportcard.com/badge/github.com/andrescosta/goico)](https://goreportcard.com/report/github.com/andrescosta/goico)
## Overview:

**Goico** is a specialized framework meticulously crafted to support the development and evolution of Jobico, the innovative job execution platform. Tailored to meet the unique requirements of Jobico, Goico offers a versatile set of features and capabilities, empowering developers to create services, expose REST or gRPC APIs, and build headless worker services seamlessly.

## Key Features:

### 1. Service Creation and API Exposure:

Goico simplifies the development of microservices within the Jobico ecosystem. Developers can effortlessly create services that expose REST or gRPC APIs, fostering a modular and scalable architecture. Whether crafting interactive interfaces or building headless worker services, Goico provides the essential foundation for diverse service types.

### 2. WASM Runtime Based on WAZERO:

A core strength of Goico lies in its WebAssembly (WASM) runtime, built on the robust foundation of WAZERO. This runtime facilitates the execution of custom logic written in any WASM-supported programming language. This empowers developers to implement dynamic and language-agnostic Jobicolets, the key components responsible for processing events and generating results.

### 3. Key/Value Embedded Database:

Goico integrates a powerful embedded database based on bbolt, offering a key/value store for efficient data management. This embedded database serves as the backbone for storing critical information, supporting the reliable and fast retrieval of data essential for the operation of Jobico.

### 4. Streaming Capabilities for Database Updates:

Goico goes beyond basic database functionality by providing advanced streaming capabilities for database updates. This feature enables real-time monitoring and reaction to changes within the embedded database, facilitating dynamic adjustments, and enhancing the responsiveness of Jobico to evolving requirements.

## Use Cases:

- **Microservices Development:**
  - Goico streamlines the creation of microservices, enabling developers to build modular components that contribute to the overall functionality of Jobico.

- **Language-Agnostic WASM Execution:**
  - With the WAZERO-based WASM runtime, Goico allows developers to implement Jobicolets in any programming language that compiles to WebAssembly, promoting flexibility and diversity in event processing.

- **Efficient Data Storage and Retrieval:**
  - The embedded database based on bbolt provides a reliable and performant key/value store, supporting the storage and retrieval of essential data within the Jobico framework.

- **Real-time Monitoring and Adaptation:**
  - Goico's streaming capabilities for database updates empower Jobico to react in real-time to changes, facilitating dynamic adjustments and ensuring responsiveness to evolving requirements.

## Conclusion:

Goico stands as a dedicated framework designed with the singular purpose of supporting the development and advancement of Jobico. Through its service creation capabilities, powerful WASM runtime, embedded database, and streaming functionalities, Goico provides the solid foundation upon which Jobico's innovative features and capabilities thrive. As Jobico evolves, Goico remains a reliable companion, empowering developers to navigate the complexities of job execution and event processing with ease and efficiency.
