use cosmwasm_std::StdError;
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

    #[error("Transfer already sent")]
    TransferAlreadySent {},

    #[error("Sender already provided the evidence")]
    EvidenceAlreadyProvided {},
}
