// __NAME__ Agent — Crypto Implementation (RC4)
//
// RC4 symmetric cipher — same algorithm as beacon_agent.
// Replace with AES/ChaCha20/etc. via protocol crypto generator.

#include "crypto.h"

#include <stdlib.h>
#include <string.h>

void RC4Init(unsigned char* key, unsigned char* S, int keyLength)
{
    int j = 0;
    unsigned char temp;

    for (int i = 0; i < 256; i++)
        S[i] = (unsigned char)i;

    for (int i = 0; i < 256; i++) {
        j = (j + S[i] + key[i % keyLength]) % 256;
        temp = S[i];
        S[i] = S[j];
        S[j] = temp;
    }
}

void RC4EncryptDecrypt(unsigned char* data, int dataLength, unsigned char* S)
{
    int i = 0, j = 0;
    unsigned char temp;

    for (int k = 0; k < dataLength; k++) {
        i = (i + 1) % 256;
        j = (j + S[i]) % 256;
        temp = S[i];
        S[i] = S[j];
        S[j] = temp;
        data[k] ^= S[(S[i] + S[j]) % 256];
    }
}

void EncryptRC4(unsigned char* data, int dataLength, unsigned char* key, int keyLength)
{
    unsigned char S[256];
    RC4Init(key, S, keyLength);
    RC4EncryptDecrypt(data, dataLength, S);
}

void DecryptRC4(unsigned char* data, int dataLength, unsigned char* key, int keyLength)
{
    EncryptRC4(data, dataLength, key, keyLength);
}

uint8_t* EncryptData(const uint8_t* data, uint32_t dataLen, const uint8_t* key, uint32_t keyLen, uint32_t* outLen)
{
    if (outLen) {
        *outLen = 0;
    }
    if (!data || dataLen == 0) {
        return nullptr;
    }

    uint8_t* out = (uint8_t*)malloc(dataLen);
    if (!out) {
        return nullptr;
    }

    memcpy(out, data, dataLen);
    if (key && keyLen > 0) {
        EncryptRC4(out, (int)dataLen, (unsigned char*)key, (int)keyLen);
    }

    if (outLen) {
        *outLen = dataLen;
    }
    return out;
}

uint8_t* DecryptData(const uint8_t* data, uint32_t dataLen, const uint8_t* key, uint32_t keyLen, uint32_t* outLen)
{
    return EncryptData(data, dataLen, key, keyLen, outLen);
}
