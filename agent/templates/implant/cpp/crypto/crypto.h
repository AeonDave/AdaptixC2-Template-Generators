// __NAME__ Agent — Crypto Implementation (RC4)
//
// Default: RC4 encryption (matches beacon_agent pattern).
// Swap this file for a different algorithm by using the protocol/crypto generator.

#pragma once

void RC4Init(unsigned char* key, unsigned char* S, int keyLength);
void RC4EncryptDecrypt(unsigned char* data, int dataLength, unsigned char* S);
void EncryptRC4(unsigned char* data, int dataLength, unsigned char* key, int keyLength);
void DecryptRC4(unsigned char* data, int dataLength, unsigned char* key, int keyLength);
