const UINT64_SAMPLE_SPACE = 1n << 64n

const normalizeBounds = (min: number, max: number): [number, number] => {
  // Invalid lower/upper bounds deliberately expand to the corresponding safe-integer endpoint.
  const normalizedMin = Number.isSafeInteger(min) ? min : Number.MIN_SAFE_INTEGER
  const normalizedMax = Number.isSafeInteger(max) ? max : Number.MAX_SAFE_INTEGER

  return normalizedMin <= normalizedMax
    ? [normalizedMin, normalizedMax]
    : [normalizedMax, normalizedMin]
}

const randomUint64 = (): bigint => {
  const words = new Uint32Array(2)
  window.crypto.getRandomValues(words)
  return (BigInt(words[0]) << 32n) | BigInt(words[1])
}

const seq = '0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('')

const RandomUtil = {
  randomIntRange(min: number, max: number): number {
    const [lower, upper] = normalizeBounds(min, max)
    if (lower === upper) {
      return lower
    }

    const range = BigInt(upper) - BigInt(lower) + 1n
    const rejectionLimit = UINT64_SAMPLE_SPACE - (UINT64_SAMPLE_SPACE % range)

    let sample: bigint
    do {
      sample = randomUint64()
    } while (sample >= rejectionLimit)

    return Number(BigInt(lower) + (sample % range))
  },
  randomInt(n: number) {
    return this.randomIntRange(0, n)
  },
  randomSeq(count: number): string {
    if (count <= 0) {
      return ''
    }
    let str = ''
    for (let i = 0; i < count; ++i) {
        str += seq[this.randomInt(seq.length - 1)]
    }
    return str
  },
  randomLowerAndNum(count: number): string {
    if (count <= 0) {
      return ''
    }
    let str = ''
    for (let i = 0; i < count; ++i) {
        str += seq[this.randomInt(35)]
    }
    return str
  },
  randomUUID(): string {
    const rng = new Uint8Array(16);
    window.crypto.getRandomValues(rng);
    rng[6] = (rng[6] & 0x0f) | 0x40;
    rng[8] = (rng[8] & 0x3f) | 0x80;
    return (
      byteToHex[rng[0]] + byteToHex[rng[1]] + byteToHex[rng[2]] + byteToHex[rng[3]] + '-' +
      byteToHex[rng[4]] + byteToHex[rng[5]] + '-' +
      byteToHex[rng[6]] + byteToHex[rng[7]] + '-' +
      byteToHex[rng[8]] + byteToHex[rng[9]] + '-' +
      byteToHex[rng[10]] + byteToHex[rng[11]] + byteToHex[rng[12]] +
      byteToHex[rng[13]] + byteToHex[rng[14]] + byteToHex[rng[15]]
    );
  },
  randomShadowsocksPassword(n: number): string {
    const array = new Uint8Array(n)
    window.crypto.getRandomValues(array)
    return btoa(String.fromCharCode(...array))
  },
  randomShortId(): string[] {
    let shortIds = new Array(24).fill('')
    for (var ii = 1; ii < 24; ii++) {
      for (var jj = 0; jj <= this.randomInt(7); jj++){
          let randomNum = this.randomInt(255)
          shortIds[ii] += ('0' + randomNum.toString(16)).slice(-2)
      }
  }
  return shortIds
  }
}

const byteToHex = Array.from(
  { length: 256 },
  (_, i) => (i + 0x100)
    .toString(16)
    .slice(1)
)

export default RandomUtil