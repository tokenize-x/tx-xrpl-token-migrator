use crate::msg;
use cosmwasm_std::{Addr, Empty, Uint128};
use cw_storage_plus::{Item, Map};

pub const OWNER: Item<Addr> = Item::new("owner");
pub const THRESHOLD: Item<u32> = Item::new("threshold");
pub const TRUSTED_ADDRESSES: Map<Addr, Empty> = Map::new("trusted_addresses");
pub const MIN_AMOUNT: Item<Uint128> = Item::new("min_amount");
pub const MAX_AMOUNT: Item<Uint128> = Item::new("max_amount");
pub const PENDING_TRANSACTIONS: Map<String, msg::Transaction> = Map::new("pending_transactions");
pub const SENT_TRANSACTIONS: Map<String, msg::Transaction> = Map::new("sent_transactions");
pub const XRPL_TOKENS: Item<Vec<msg::XRPLToken>> = Item::new("xrpl_tokens");
