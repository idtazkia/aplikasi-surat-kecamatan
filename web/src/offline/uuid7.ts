// UUIDv7 client-side generator.
//
// Format: xxxxxxxx-xxxx-7xxx-yxxx-xxxxxxxxxxxx
//   - First 48 bits = unix timestamp millisecond (time-ordered prefix)
//   - Next 4 bits = version "0111" (= 7)
//   - Next 12 bits = random (rand_a)
//   - Next 2 bits = variant "10"
//   - Remaining 62 bits = random (rand_b)
//
// Untuk offline-first: client-generated UUIDv7 menghindari create-conflict
// antar klien — 2 klien offline bisa create entity dengan ID berbeda (random
// component) tanpa koordinasi server. Saat sync, server terima keduanya tanpa
// renumber. ID time-ordered → cocok untuk PK B-tree append-only insert tanpa
// fragmentation.
//
// Concept anchor utama di backend (internal/uuid7).
export function uuidv7(): string {
  const now = Date.now(); // ms

  // 48-bit timestamp sebagai 12-char hex
  const tsHex = now.toString(16).padStart(12, "0");
  const tsPart1 = tsHex.slice(0, 8); // 32 bits
  const tsPart2 = tsHex.slice(8, 12); // 16 bits

  // 80 bit random buffer untuk version+rand_a + variant+rand_b + tail
  const rand = new Uint8Array(10);
  if (typeof crypto !== "undefined" && crypto.getRandomValues) {
    crypto.getRandomValues(rand);
  } else {
    // Math.random fallback — non-secure tapi OK untuk dev/test
    for (let i = 0; i < rand.length; i++) {
      rand[i] = Math.floor(Math.random() * 256);
    }
  }

  // Force version 7: high nibble byte[0] = 0x7
  rand[0] = (rand[0] & 0x0f) | 0x70;
  // Force variant 10 (RFC 4122): high 2 bits byte[2] = 0b10
  rand[2] = (rand[2] & 0x3f) | 0x80;

  const hex2 = (n: number) => n.toString(16).padStart(2, "0");
  const bytesHex = (off: number, len: number): string => {
    let s = "";
    for (let i = 0; i < len; i++) s += hex2(rand[off + i]);
    return s;
  };

  return `${tsPart1}-${tsPart2}-${bytesHex(0, 2)}-${bytesHex(2, 2)}-${bytesHex(4, 6)}`;
}
