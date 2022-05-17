use tiny_keccak::{Hasher, Keccak};

pub fn keccak256(data: &[u8]) -> [u8; 32] {
    let mut hasher = Keccak::v256();
    let mut out: [u8; 32] = [0; 32];
    hasher.update(data);
    hasher.finalize(&mut out);
    out
}
