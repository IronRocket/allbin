function Unit8ToBase64(uint8Array: Uint8Array<ArrayBuffer>): string {
  let binary = "";
  for (let i = 0; i < uint8Array.length; i++) {
    binary += String.fromCharCode(uint8Array[i]);
  }
  return window.btoa(binary);
}

export async function aesKeyFromPassword(password: string) {
  // Turn password into 32 bytes (AES-256)
  const raw = new TextEncoder().encode(password);
  const keyBytes = new Uint8Array(32);
  keyBytes.set(raw.slice(0, 32)); // pad or truncate

  return crypto.subtle.importKey("raw", keyBytes, { name: "AES-GCM" }, false, [
    "encrypt",
    "decrypt",
  ]);
}

export async function decryptAES(data: string, aes: CryptoKey) {
  const [cipherTextBase64, ivBase64] = data.split("iv:");

  const cipherText = Uint8Array.from(atob(cipherTextBase64), (c) =>
    c.charCodeAt(0),
  );
  const iv = Uint8Array.from(atob(ivBase64), (c) => c.charCodeAt(0));

  const decryptedBuffer = await crypto.subtle.decrypt(
    {
      name: "AES-GCM",
      iv: iv,
    },
    aes, // assumed to be a CryptoKey already
    cipherText,
  );

  return new TextDecoder().decode(decryptedBuffer);
}

export async function transformRawPublicKey(text: string) {
  const raw_key = window.atob(text.substring(31, text.length - 29));
  const bytes = new Uint8Array(raw_key.length);
  for (let i = 0, strLen = raw_key.length; i < strLen; i++) {
    bytes[i] = raw_key.charCodeAt(i);
  }
  const serverPublickey = await crypto.subtle.importKey(
    "spki",
    bytes.buffer,
    {
      name: "RSA-OAEP",
      hash: "SHA-256",
    },
    true,
    ["encrypt"],
  );

  return serverPublickey;
}

export async function generateKeys() {
  let aes = await crypto.subtle.generateKey(
    {
      name: "AES-GCM",
      length: 256,
    },
    true,
    ["encrypt", "decrypt"],
  );
  let aesRaw = await crypto.subtle.exportKey("raw", aes);
  return { aes, aesRaw };
}

// IV is stored in the first 12 indicies of the array.
// Everything after that is the encrypted data
export function encryptedIVCombine(
  encrypted: ArrayBuffer,
  iv: Uint8Array<ArrayBuffer>,
): Uint8Array<ArrayBuffer> {
  const wrapper = new Uint8Array(iv.length + encrypted.byteLength);
  wrapper.set(iv);
  wrapper.set(new Uint8Array(encrypted), iv.length);

  return wrapper;
}

export function encryptedIVSplit(
  mixture: Uint8Array<ArrayBuffer>,
): [Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>] {
  const iv = mixture.slice(0, 12);
  const encryptedText = mixture.slice(12);

  return [iv, encryptedText];
}

export async function encryptWithPassword(
  password: string,
  blob: BufferSource,
): Promise<Uint8Array<ArrayBuffer>> {
  const key = await aesKeyFromPassword(password);
  const iv = crypto.getRandomValues(new Uint8Array(12));

  const encrypted = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv },
    key,
    blob,
  );

  return encryptedIVCombine(encrypted, iv);
}

export async function decryptWithPassword(
  password: string,
  mixture: Uint8Array<ArrayBuffer>,
): Promise<ArrayBuffer> {
  const key = await aesKeyFromPassword(password);

  let [iv, encryptedText] = encryptedIVSplit(mixture);

  const decrypted = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv },
    key,
    encryptedText,
  );

  return decrypted;
}

export async function base64encryptWithPassword(
  password: string,
  blob: BufferSource,
): Promise<string> {
  let encrypted = await encryptWithPassword(password, blob);
  return Unit8ToBase64(encrypted);
}
