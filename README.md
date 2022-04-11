# Jina

[![Jina logo](/assets/images/jina_logo.svg)](https://youtu.be/19qQuqAtNQE "Jina demo video")

## Demo

Demo video: https://www.youtube.com/watch?v=19qQuqAtNQE 

[Presentation](/assets/presentation.pptx)

## About Jina

Jina dapp is an NFT collateralizing app on Algorand.
Use any NFT you hold in Algorand as leverage for borrowing stable coin in Jina.
Leveraged NFT remains locked in your (*borrower*) address until the full loan amount is repaid, which is a **pure non-custodial** transaction.
Assets(NFT) that require KYC or special permissions might benefit from this *pure non-custodial protocol*.

With a 3% return for liquidity providers that supply liquidity to the protocol, Jina becomes the go-to site for delivering liquidity for NFTs.
Liquidity providers will keep their liquidity asset in their account until a borrower uses it, at which point the protocol will reward them with an I-O-U stable-coin token plus 3% of the lent amount.

## Using Jina dapp

First step is to optin to the smartcontract.

### Optin to Jina

#### As NFT creator

Transfer your NFT's admin address to Jina.
This will make your NFT leverageable for taking loan in Jina dapp.
* Set manager and freeze admin address to Jina smartcontract
* Set clawback to liquidator smartcontract

![Screenshot transfer NFT admin addresses](/assets/images/acfg_to_jina.png)

#### As a liquidity provider

Optin to the I-O-U token of jina dapp **JUSD**, that has 1:1 value with USDCa.

[![Optin to JUSD](/assets/images/optin_asa.jpg)](https://youtu.be/19qQuqAtNQE?t=95 "Optin to JUSD")

### Providing Liquidity

Choose which NFTs can borrow from your account.
* Set maximum amount you are willing to lend.
* Set expiration date for aggrement.

[![Providing liquidity](/assets/images/lend.png)](https://youtu.be/19qQuqAtNQE?t=51 "Stake your USDC")

### Leveraging NFT

Use your NFT as collateral, to borrow USDCa stablecoin.
* Set which NFT you want to collateralize
* Set amount of collateral
* Request loan
You'll get requested loan amount in USDCa and your NFT will be locked.

[![Leverage NFT](/assets/images/borrow.png)](https://youtu.be/19qQuqAtNQE?t=114 "Borrow in Jina")

### Repaying loan

Send USDCa to Jina contract.
Your loan amount state will be decremented by sent repaid amount.

[![Repay loan](/assets/images/repay.png)](https://youtu.be/19qQuqAtNQE?t=149 "Repaying loan")

If you pay the full loan amount, your collateral assets will be unfrozen.

### Claming USDCa

Send JUSD(I-O-U token of Jina contract) to Jina contract.
You'll receive a 1:1 USDCa for the JUSD you send.

[![Claim USDCa](/assets/images/claim.png)](https://youtu.be/19qQuqAtNQE?t=95 "Claim")

## Future

We plan on releasing this project on mainnet.

## Building Jina Locally

1. Go to [Jina frontend](https://github.com/adapole/jina_frontend) and deploy the web interface.
2. [Optin](https://github.com/Adg0/Jina#optin-to-jina) to Jina contract.

## Techincal info

* NFTs used as collateral are frozen in account, only when account takes out loan.
* Frozen NFTs are unfrozen when full loan is paid back.
* There is no interest rate for borrowing USDCa.
* A 3% fee is paid to take out loan.
* Lenders sign a delegated logic signature to allow any account to withdraw USDCa that fullfill the following:
	1. Calls Jina contract
	2. Withdraws atmost staked amount
* Any account that holds JUSD can claim 1:1 USDCa by sending the JUSD to Jina contract.
* Borrower can borrow from upto 4 lenders
* Liquidation

How liquidation happens?

[![Liquidation anim](/assets/images/liquidate-anim.png)](https://youtu.be/19qQuqAtNQE?t=162 "Liquidation")

* Specify the addresss to liquidate
* Pay 95% of collateral's value to Jina contract
* Set an account that will receive the liquidated asset
* You'll be sent the collateral to the address you specified

[![Liquidation](/assets/images/liquidate.png)](https://youtu.be/19qQuqAtNQE?t=189 "After liquidation")


### Smartcontract

There are two smartcontracts that power Jina dapp.

1. Jina Contract
Jina contract holds the state machine and locks/unlocks NFT in account (freezes/unfreezes  NFT) .
State machine, tracks:
	* `xids` tracks which NFT is used as collateral
	* `camt` tracks how much collateral is used for loan
	* `lamt` tracks how much loan is borrowed
	* `aamt` tracks how much loan is available from lender address

2. Liquidator Contract
Liquidator contract reads current price of NFT from oracle and if loan is more than 90% of collateral it liquidates the NFT locked.
	* liquidator contract is the clawback address of leveragable NFTs on Jina.
	* after liquidation completes the remainig asset is unfrozen. This is possible by AVM 1.1 (contract to contract call). Liquidator contract calls Jina contract to unfreeze the asset.

# Contact
Discord @1egen#0803
Discord @3spear#9556
