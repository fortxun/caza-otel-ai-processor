/** Exported memory */
export declare const memory: WebAssembly.Memory;
/** Exported table */
export declare const table: WebAssembly.Table;
// Exported runtime interface
export declare function __new(size: number, id: number): number;
export declare function __pin(ptr: number): number;
export declare function __unpin(ptr: number): void;
export declare function __collect(): void;
export declare const __rtti_base: number;
/**
 * error-classifier/assembly/index/classify_error
 * @param inputJson `~lib/string/String`
 * @returns `~lib/string/String`
 */
export declare function classify_error(inputJson: string): string;
