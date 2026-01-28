import {
    Connection,
    PublicKey,
    Transaction,
    TransactionInstruction,
    SystemProgram,
    SYSVAR_RENT_PUBKEY,
    Keypair,
    AccountMeta,
  } from '@solana/web3.js';
  import {
    TOKEN_PROGRAM_ID,
    ASSOCIATED_TOKEN_PROGRAM_ID,
    getAssociatedTokenAddress,
    createAssociatedTokenAccountInstruction,
  } from '@solana/spl-token';
  import * as borsh from '@coral-xyz/borsh';
  import BN from 'bn.js';
  
  export interface PumpFunInternalPoolStat {
    timestamp: number;
    mint: string;
    feeRate: number;
    unknownData: number;
    virtualTokenReserves: number;
    virtualSolReserves: number;
    realTokenReserves: number;
    realSolReserves: number;
    tokenTotalSupply: number;
    complete: boolean;
    creator: string;
    price: number;
    feeRecipient: string;
    bondingCurvePDA: string;
    associatedBondingCurve: string;
    creatorVaultPDA: string;
  }
  
  
  // PumpFun 程序地址
  export const PUMP_FUN_PROGRAM_ID = new PublicKey('6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P');
  export const EVENT_AUTHORITY = new PublicKey('Ce6TQqeHC9p8KetsN6JsjHK7UTZk7nasjjnr7XxXp9F1');
  export const MPL_TOKEN_METADATA_PROGRAM_ID = new PublicKey('metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s');
  
  // 指令判别器
  const INSTRUCTION_DISCRIMINATORS = {
    initialize: Buffer.from([175, 175, 109, 31, 13, 152, 155, 237]),
    setParams: Buffer.from([165, 31, 134, 53, 189, 180, 130, 255]),
    create: Buffer.from([24, 30, 200, 40, 5, 28, 7, 119]),
    buy: Buffer.from([102, 6, 61, 18, 1, 218, 235, 234]),
    sell: Buffer.from([51, 230, 133, 164, 1, 127, 131, 173]),
    withdraw: Buffer.from([183, 18, 70, 156, 148, 109, 161, 34]),
  };
  
  const BONDING_LAYOUT = borsh.struct([
    borsh.u64('unknown_data'),   // 1
    borsh.u64('virtual_token_reserves'),   // 1
    borsh.u64('virtual_sol_reserves'),     // 2
    borsh.u64('real_token_reserves'),      // 3
    borsh.u64('real_sol_reserves'),        // 4
    borsh.u64('token_total_supply'),       // 5
    borsh.bool('complete'),                // 6
    borsh.publicKey('creator'),            // 7
  ]);
  
  export async function getPumpFunInternalPoolStat(connection: Connection, mint: PublicKey, feeRate: number, feeRecipient: PublicKey): Promise<PumpFunInternalPoolStat> {
    const [bondingCurvePDA] = getBondingCurvePDA(mint);
    const accountInfo = await connection.getAccountInfo(bondingCurvePDA);
  
    if (!accountInfo) throw new Error('Bonding PDA not found');
  
    let state = BONDING_LAYOUT.decode(accountInfo.data);
    const virtual_token_reserves_number = Number(state.virtual_token_reserves.toString());
    const virtual_sol_reserves_number = Number(state.virtual_sol_reserves.toString());
    const price = (virtual_sol_reserves_number / 1e9) / (virtual_token_reserves_number / 1e6);
  
    const associatedBondingCurve = await getAssociatedTokenAddress(
      mint,
      bondingCurvePDA,
      true
    );
  
    const [creatorVaultPDA] = getCreatorVaultPDA(new PublicKey(state.creator.toBase58()));
  
    const pumpFunInternalPoolStat: PumpFunInternalPoolStat = {
      timestamp: Date.now(),
      mint: mint.toBase58(),
      feeRate: 0.01,
      unknownData: Number(state.unknown_data.toString()),
      virtualTokenReserves: virtual_token_reserves_number,
      virtualSolReserves: virtual_sol_reserves_number,
      realTokenReserves: Number(state.real_token_reserves.toString()),
      realSolReserves: Number(state.real_sol_reserves.toString()),
      tokenTotalSupply: Number(state.token_total_supply.toString()),
      complete: state.complete,
      creator: state.creator.toBase58(),
      price: price,
      feeRecipient: feeRecipient.toBase58(),
      bondingCurvePDA: bondingCurvePDA.toBase58(),
      associatedBondingCurve: associatedBondingCurve.toBase58(),
      creatorVaultPDA: creatorVaultPDA.toBase58(),
    };
  
    console.log('✅ Pump Bonding State:');
    // console.log(`[pumpFunInternalPoolStat] json: ${JSON.stringify(pumpFunInternalPoolStat)}`);
    console.table(pumpFunInternalPoolStat);
    return pumpFunInternalPoolStat;
  }
  
  // PDA 种子
  const SEEDS = {
    GLOBAL: Buffer.from('global'),
    MINT_AUTHORITY: Buffer.from('mint-authority'),
    BONDING_CURVE: Buffer.from('bonding-curve'),
    CREATOR_VAULT: Buffer.from('creator-vault'),
  };
  
  /**
   * 获取全局状态 PDA
   */
  export function getGlobalPDA(): [PublicKey, number] {
    return PublicKey.findProgramAddressSync(
      [SEEDS.GLOBAL],
      PUMP_FUN_PROGRAM_ID
    );
  }
  
  /** 
   * 获取创建者 vault PDA
   */
  export function getCreatorVaultPDA(user: PublicKey): [PublicKey, number] {
    return PublicKey.findProgramAddressSync(
      [
        SEEDS.CREATOR_VAULT,
        user.toBuffer()
      ],
      PUMP_FUN_PROGRAM_ID
    );
  }
  
  /**
   * 获取铸币权限 PDA
   */
  export function getMintAuthorityPDA(): [PublicKey, number] {
    return PublicKey.findProgramAddressSync(
      [SEEDS.MINT_AUTHORITY],
      PUMP_FUN_PROGRAM_ID
    );
  }
  
  /**
   * 获取联合曲线 PDA
   */
  export function getBondingCurvePDA(mint: PublicKey): [PublicKey, number] {
    return PublicKey.findProgramAddressSync(
      [SEEDS.BONDING_CURVE, mint.toBuffer()],
      PUMP_FUN_PROGRAM_ID
    );
  }
  
  /**
   * 获取代币元数据 PDA
   */
  export function getMetadataPDA(mint: PublicKey): [PublicKey, number] {
    return PublicKey.findProgramAddressSync(
      [
        Buffer.from('metadata'),
        MPL_TOKEN_METADATA_PROGRAM_ID.toBuffer(),
        mint.toBuffer(),
      ],
      MPL_TOKEN_METADATA_PROGRAM_ID
    );
  }
  
  /**
   * 序列化字符串
   */
  function serializeString(str: string): Buffer {
    const strBuffer = Buffer.from(str, 'utf8');
    const lengthBuffer = Buffer.alloc(4);
    lengthBuffer.writeUInt32LE(strBuffer.length, 0);
    return Buffer.concat([lengthBuffer, strBuffer]);
  }
  
  /**
   * 序列化 u64
   */
  function serializeU64(value: BN | number): Buffer {
    const bn = typeof value === 'number' ? new BN(value) : value;
    const buffer = Buffer.alloc(8);
    bn.toArrayLike(Buffer, 'le', 8).copy(buffer);
    return buffer;
  }
  
  /**
   * 序列化公钥
   */
  function serializePubkey(pubkey: PublicKey): Buffer {
    return pubkey.toBuffer();
  }
  
  /**
   * 初始化指令
   */
  export function createInitializeInstruction(
    user: PublicKey
  ): TransactionInstruction {
    const [globalPDA] = getGlobalPDA();
  
    const accounts: AccountMeta[] = [
      { pubkey: globalPDA, isSigner: false, isWritable: true },
      { pubkey: user, isSigner: true, isWritable: true },
      { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
    ];
  
    return new TransactionInstruction({
      keys: accounts,
      programId: PUMP_FUN_PROGRAM_ID,
      data: INSTRUCTION_DISCRIMINATORS.initialize,
    });
  }
  
  /**
   * 设置参数指令
   */
  export function createSetParamsInstruction(
    user: PublicKey,
    feeRecipient: PublicKey,
    initialVirtualTokenReserves: BN,
    initialVirtualSolReserves: BN,
    initialRealTokenReserves: BN,
    tokenTotalSupply: BN,
    feeBasisPoints: BN
  ): TransactionInstruction {
    const [globalPDA] = getGlobalPDA();
  
    const accounts: AccountMeta[] = [
      { pubkey: globalPDA, isSigner: false, isWritable: true },
      { pubkey: user, isSigner: true, isWritable: true },
      { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
      { pubkey: EVENT_AUTHORITY, isSigner: false, isWritable: false },
      { pubkey: PUMP_FUN_PROGRAM_ID, isSigner: false, isWritable: false },
    ];
  
    const data = Buffer.concat([
      INSTRUCTION_DISCRIMINATORS.setParams,
      serializePubkey(feeRecipient),
      serializeU64(initialVirtualTokenReserves),
      serializeU64(initialVirtualSolReserves),
      serializeU64(initialRealTokenReserves),
      serializeU64(tokenTotalSupply),
      serializeU64(feeBasisPoints),
    ]);
  
    return new TransactionInstruction({
      keys: accounts,
      programId: PUMP_FUN_PROGRAM_ID,
      data,
    });
  }
  
  /**
   * 创建代币指令
   */
  export async function createCreateInstruction(
    mint: PublicKey,
    user: PublicKey,
    name: string,
    symbol: string,
    uri: string,
    creator: PublicKey
  ): Promise<TransactionInstruction> {
    const [globalPDA] = getGlobalPDA();
    const [mintAuthorityPDA] = getMintAuthorityPDA();
    const [bondingCurvePDA] = getBondingCurvePDA(mint);
    const [metadataPDA] = getMetadataPDA(mint);
    
    const associatedBondingCurve = await getAssociatedTokenAddress(
      mint,
      bondingCurvePDA,
      true
    );
  
    const accounts: AccountMeta[] = [
      { pubkey: mint, isSigner: true, isWritable: true },
      { pubkey: mintAuthorityPDA, isSigner: false, isWritable: false },
      { pubkey: bondingCurvePDA, isSigner: false, isWritable: true },
      { pubkey: associatedBondingCurve, isSigner: false, isWritable: true },
      { pubkey: globalPDA, isSigner: false, isWritable: false },
      { pubkey: MPL_TOKEN_METADATA_PROGRAM_ID, isSigner: false, isWritable: false },
      { pubkey: metadataPDA, isSigner: false, isWritable: true },
      { pubkey: user, isSigner: true, isWritable: true },
      { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
      { pubkey: TOKEN_PROGRAM_ID, isSigner: false, isWritable: false },
      { pubkey: ASSOCIATED_TOKEN_PROGRAM_ID, isSigner: false, isWritable: false },
      { pubkey: SYSVAR_RENT_PUBKEY, isSigner: false, isWritable: false },
      { pubkey: EVENT_AUTHORITY, isSigner: false, isWritable: false },
      { pubkey: PUMP_FUN_PROGRAM_ID, isSigner: false, isWritable: false },
    ];
  
    const data = Buffer.concat([
      INSTRUCTION_DISCRIMINATORS.create,
      serializeString(name),
      serializeString(symbol),
      serializeString(uri),
      serializePubkey(creator),
    ]);
  
    return new TransactionInstruction({
      keys: accounts,
      programId: PUMP_FUN_PROGRAM_ID,
      data,
    });
  }
  
  /**
   * 购买代币指令
   */
  export async function createBuyInstruction(
    mint: PublicKey,
    bondingCurvePDA: PublicKey,
    associatedBondingCurve: PublicKey,
    associatedUser: PublicKey,
    creatorVaultPDA: PublicKey,
    user: PublicKey,
    feeRecipient: PublicKey,
    amount: BN,
    maxSolCost: BN
  ): Promise<TransactionInstruction> {
    const [globalPDA] = getGlobalPDA();
    
    // const associatedBondingCurve = await getAssociatedTokenAddress(
    //   mint,
    //   bondingCurvePDA,
    //   true
    // );
    
    // const associatedUser = await getAssociatedTokenAddress(mint, user);
  
    const accounts: AccountMeta[] = [
      { pubkey: globalPDA, isSigner: false, isWritable: false },
      { pubkey: feeRecipient, isSigner: false, isWritable: true },
      { pubkey: mint, isSigner: false, isWritable: false },
      { pubkey: bondingCurvePDA, isSigner: false, isWritable: true },
      { pubkey: associatedBondingCurve, isSigner: false, isWritable: true },
      { pubkey: associatedUser, isSigner: false, isWritable: true },
      { pubkey: user, isSigner: true, isWritable: true },
      { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
      { pubkey: TOKEN_PROGRAM_ID, isSigner: false, isWritable: false },
      { pubkey: creatorVaultPDA, isSigner: false, isWritable: true },
      { pubkey: EVENT_AUTHORITY, isSigner: false, isWritable: false },
      { pubkey: PUMP_FUN_PROGRAM_ID, isSigner: false, isWritable: false },
    ];
    // console.table(accounts);
    const data = Buffer.concat([
      INSTRUCTION_DISCRIMINATORS.buy,
      serializeU64(amount),
      serializeU64(maxSolCost),
    ]);
  
    return new TransactionInstruction({
      keys: accounts,
      programId: PUMP_FUN_PROGRAM_ID,
      data,
    });
  }
  
  /**
   * 出售代币指令
   */
  export async function createSellInstruction(
    mint: PublicKey,
    bondingCurvePDA: PublicKey,
    associatedBondingCurve: PublicKey,
    associatedUser: PublicKey,
    creatorVaultPDA: PublicKey,
    user: PublicKey,
    feeRecipient: PublicKey,
    amount: BN,
    minSolOutput: BN
  ): Promise<TransactionInstruction> {
    const [globalPDA] = getGlobalPDA();
    
    // const associatedBondingCurve = await getAssociatedTokenAddress(
    //   mint,
    //   bondingCurvePDA,
    //   true
    // );
    
    // const associatedUser = await getAssociatedTokenAddress(mint, user);
  
    const accounts: AccountMeta[] = [
      { pubkey: globalPDA, isSigner: false, isWritable: false },
      { pubkey: feeRecipient, isSigner: false, isWritable: true },
      { pubkey: mint, isSigner: false, isWritable: false },
      { pubkey: bondingCurvePDA, isSigner: false, isWritable: true },
      { pubkey: associatedBondingCurve, isSigner: false, isWritable: true },
      { pubkey: associatedUser, isSigner: false, isWritable: true },
      { pubkey: user, isSigner: true, isWritable: true },
      { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
      { pubkey: creatorVaultPDA, isSigner: false, isWritable: true },
      { pubkey: TOKEN_PROGRAM_ID, isSigner: false, isWritable: false },
      { pubkey: EVENT_AUTHORITY, isSigner: false, isWritable: false },
      { pubkey: PUMP_FUN_PROGRAM_ID, isSigner: false, isWritable: false },
    ];
    // console.table(accounts);
  
    const data = Buffer.concat([
      INSTRUCTION_DISCRIMINATORS.sell,
      serializeU64(amount),
      serializeU64(minSolOutput),
    ]);
  
    return new TransactionInstruction({
      keys: accounts,
      programId: PUMP_FUN_PROGRAM_ID,
      data,
    });
  }
  
  /**
   * 提取流动性指令
   */
  export async function createWithdrawInstruction(
    mint: PublicKey,
    user: PublicKey,
    lastWithdraw: PublicKey
  ): Promise<TransactionInstruction> {
    const [globalPDA] = getGlobalPDA();
    const [bondingCurvePDA] = getBondingCurvePDA(mint);
    
    const associatedBondingCurve = await getAssociatedTokenAddress(
      mint,
      bondingCurvePDA,
      true
    );
    
    const associatedUser = await getAssociatedTokenAddress(mint, user);
  
    const accounts: AccountMeta[] = [
      { pubkey: globalPDA, isSigner: false, isWritable: false },
      { pubkey: lastWithdraw, isSigner: false, isWritable: true },
      { pubkey: mint, isSigner: false, isWritable: false },
      { pubkey: bondingCurvePDA, isSigner: false, isWritable: true },
      { pubkey: associatedBondingCurve, isSigner: false, isWritable: true },
      { pubkey: associatedUser, isSigner: false, isWritable: true },
      { pubkey: user, isSigner: true, isWritable: true },
      { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
      { pubkey: TOKEN_PROGRAM_ID, isSigner: false, isWritable: false },
      { pubkey: SYSVAR_RENT_PUBKEY, isSigner: false, isWritable: false },
      { pubkey: EVENT_AUTHORITY, isSigner: false, isWritable: false },
      { pubkey: PUMP_FUN_PROGRAM_ID, isSigner: false, isWritable: false },
    ];
  
    return new TransactionInstruction({
      keys: accounts,
      programId: PUMP_FUN_PROGRAM_ID,
      data: INSTRUCTION_DISCRIMINATORS.withdraw,
    });
  }