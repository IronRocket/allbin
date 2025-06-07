export async function base64decryptWithPassword(
  password: string,
  base64: string,
) {
  let uint8 = base64ToUint8(base64);

  let decrypted = await decryptWithPassword(password, uint8);
  let decryptedText = new TextDecoder().decode(decrypted);
  return decryptedText;
}

function base64ToUint8(base64: string): Uint8Array<ArrayBuffer> {
  const binary = window.atob(base64);
  const len = binary.length;
  const bytes = new Uint8Array(len);
  for (let i = 0; i < len; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

async function aesKeyFromPassword(password: string) {
  // Turn password into 32 bytes (AES-256)
  const raw = new TextEncoder().encode(password);
  const keyBytes = new Uint8Array(32);
  keyBytes.set(raw.slice(0, 32)); // pad or truncate

  return crypto.subtle.importKey("raw", keyBytes, { name: "AES-GCM" }, false, [
    "encrypt",
    "decrypt",
  ]);
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

function encryptedIVSplit(
  mixture: Uint8Array<ArrayBuffer>,
): [Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>] {
  const iv = mixture.slice(0, 12);
  const encryptedText = mixture.slice(12);

  return [iv, encryptedText];
}
