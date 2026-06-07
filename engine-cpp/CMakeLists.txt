cmake_minimum_required(VERSION 3.16)
project(aspira_engine VERSION 2.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_FLAGS_RELEASE "-O3 -march=native -DNDEBUG")
set(CMAKE_CXX_FLAGS_DEBUG "-g -O0 -fsanitize=address,undefined")

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

set(SOURCES
    src/main.cpp
    src/Engine.cpp
    src/Ledger.cpp
    src/CommandQueue.cpp
    src/WAL.cpp
    src/Publisher.cpp
)

add_executable(aspira_engine ${SOURCES})

target_include_directories(aspira_engine PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(aspira_engine PRIVATE
    OpenSSL::SSL
    OpenSSL::Crypto
    Threads::Threads
)

# Installation
install(TARGETS aspira_engine RUNTIME DESTINATION bin)
