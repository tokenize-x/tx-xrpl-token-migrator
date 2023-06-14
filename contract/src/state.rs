use crate::msg;
use cosmwasm_std::{Addr, Empty};
use cw_storage_plus::{Item, Map};

pub const OWNER: Item<Addr> = Item::new("owner");
pub const THRESHOLD: Item<u32> = Item::new("threshold");
pub const TRUSTED_ADDRESSES: Map<Addr, Empty> = Map::new("trusted_addresses");
pub const PENDING_TRANSACTIONS: Map<String, msg::Transaction> = Map::new("pending_transactions");
pub const SENT_TRANSACTIONS: Map<String, msg::Transaction> = Map::new("sent_transactions");
