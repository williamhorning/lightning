if (import.meta.main) {
    await import('./cli.ts');
}

export * from './lightning.ts';
export * from './structures/mod.ts';
