use cosmwasm_std::StdError;
use cw_utils::PaymentError;
use thiserror::Error;

#[derive(Error, Debug, PartialEq)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("Unauthorized")]
    Unauthorized {},

    #[error("Custom Error val: {val:?}")]
    CustomError { val: String },

    #[error("The threshold must be greater than zero and lower or equal to the trusted addresses number")]
    InvalidThreshold {},

    #[error("Duplicated trusted address")]
    DuplicatedTrustedAddress {},

    #[error("Duplicated XRPL token")]
    DuplicatedXRPLToken {},

    #[error("Duplicated BSC token: {bridge_address}")]
    DuplicatedBscToken { bridge_address: String },

    #[error("Invalid XRPL currency format: {reason}")]
    InvalidXRPLCurrency { reason: String },

    #[error("Invalid XRPL issuer format: {reason}")]
    InvalidXRPLIssuer { reason: String },

    #[error("Invalid multiplier format: {reason}")]
    InvalidMultiplier { reason: String },

    #[error("Transfer already sent")]
    TransferAlreadySent {},

    #[error("Sender already provided the evidence")]
    EvidenceAlreadyProvided {},

    #[error("Transaction not found")]
    TransactionNotFound {},

    #[error("Transaction not confirmed")]
    TransactionNotConfirmed {},

    #[error("The amount is too low")]
    LowAmount {},

    #[error("Funds mismatch")]
    FundsMismatch {},

    #[error("{0}")]
    Payment(#[from] PaymentError),
}
