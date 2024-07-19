const ElementToBigInt = (e /* *Element*/) => {
  let i = 0n;
  i <<= 51n - 32n;
  i += BigInt(e.l4.$high);
  i <<= 32n;
  i += BigInt(e.l4.$low);

  i <<= 51n - 32n;
  i += BigInt(e.l3.$high);
  i <<= 32n;
  i += BigInt(e.l3.$low);

  i <<= 51n - 32n;
  i += BigInt(e.l2.$high);
  i <<= 32n;
  i += BigInt(e.l2.$low);

  i <<= 51n - 32n;
  i += BigInt(e.l1.$high);
  i <<= 32n;
  i += BigInt(e.l1.$low);

  i <<= 51n - 32n;
  i += BigInt(e.l0.$high);
  i <<= 32n;
  i += BigInt(e.l0.$low);
  return i;
};

const BigIntToElement = (i /* bigint*/, e /* *Element*/) => {
  let low, high;
  low = Number(BigInt.asUintN(32, i)) >>> 0;
  i >>= 32n;
  high = Number(BigInt.asUintN(51 - 32, i)) >>> 0;
  i >>= 51n - 32n;
  e.l0 = new $Uint64(high, low);

  low = Number(BigInt.asUintN(32, i)) >>> 0;
  i >>= 32n;
  high = Number(BigInt.asUintN(51 - 32, i)) >>> 0;
  i >>= 51n - 32n;
  e.l1 = new $Uint64(high, low);

  low = Number(BigInt.asUintN(32, i)) >>> 0;
  i >>= 32n;
  high = Number(BigInt.asUintN(51 - 32, i)) >>> 0;
  i >>= 51n - 32n;
  e.l2 = new $Uint64(high, low);

  low = Number(BigInt.asUintN(32, i)) >>> 0;
  i >>= 32n;
  high = Number(BigInt.asUintN(51 - 32, i)) >>> 0;
  i >>= 51n - 32n;
  e.l3 = new $Uint64(high, low);

  low = Number(BigInt.asUintN(32, i)) >>> 0;
  i >>= 32n;
  high = Number(BigInt.asUintN(51 - 32, i)) >>> 0;
  i >>= 51n - 32n;
  e.l4 = new $Uint64(high, low);
};

const mod = (2n ** 255n) - 19n;

$linknames["crypto/internal/edwards25519/field.feMulGeneric_js"] = function (
  v,
  x,
  y /* *Element*/
) {
  const bx = ElementToBigInt(x);
  const by = ElementToBigInt(y);
  BigIntToElement(bx, x);
  BigIntToElement(ElementToBigInt(x), x);
  const bv = (bx * by) % mod;
  BigIntToElement(bv, v);
};
