// __NAME__ Agent — Crypto Implementation (RC4)
//
// Default: RC4 encryption (matches beacon_agent pattern).
// Swap this file for a different algorithm by using the protocol/crypto generator.

#pragma once

#include <stdint.h>

void RC4Init(unsigned char* key, unsigned char* S, int keyLength);
void RC4EncryptDecrypt(unsigned char* data, int dataLength, unsigned char* S);
void EncryptRC4(unsigned char* data, int dataLength, unsigned char* key, int keyLength);
void DecryptRC4(unsigned char* data, int dataLength, unsigned char* key, int keyLength);

// Generic crypto shim used by shared transport code.
// Protocol overlays may replace these with protocol-specific implementations.
uint8_t* EncryptData(const uint8_t* data, uint32_t dataLen, const uint8_t* key, uint32_t keyLen, uint32_t* outLen);
uint8_t* DecryptData(const uint8_t* data, uint32_t dataLen, const uint8_t* key, uint32_t keyLen, uint32_t* outLen);
