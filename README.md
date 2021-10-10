### x448 - curve448 ECDH
#### Yawning Angel (yawning at schwanenlied dot me)

See: https://www.rfc-editor.org/rfc/rfc7748.txt

If you're familiar with how to use golang.org/x/crypto/curve25519, you will be
right at home with using x448, since the functions are the same.  Generate a
random secret key, ScalarBaseMult() to get the public key, etc etc etc.

On 64-bit targets the underlying field arithmetic uses output taken from
the fiat-crypto project.  The 32-bit version and the actual ECDH implementation
are based off Michael Hamburg's portable x448 implementation.

Notes:

 * The build-tag system used to determine which version to build is sub-optimal
   in the extreme (https://github.com/golang/go/issues/33388)

 * Unless your system has a constant-time `32x32=64-bit` or `64x64=128-bit`
   multiply (depending on backend), this is unsafe to use.  Most modern CPUs
   provide something adequate, with the notable exception of WASM.

 * As a matter of taste, and because it is prefered when implementing Noise,
   the optional all-zero check is not done.
