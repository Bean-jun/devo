const esbuild = require('esbuild')

const production = process.argv.includes('--production')
const watch = process.argv.includes('--watch')

async function main() {
  const ctx = await esbuild.context({
    entryPoints: ['src/extension.ts'],
    bundle: true,
    format: 'cjs',
    minify: production,
    sourcemap: !production,
    sourcesContent: false,
    platform: 'node',
    outfile: 'dist/extension.js',
    external: ['vscode'],
    logLevel: 'silent',
    plugins: [
      {
        name: 'vscode-plugin',
        setup(build) {
          build.onResolve({ filter: /^vscode$/ }, (args) => {
            return { path: args.path, external: true }
          })
        },
      },
    ],
  })

  if (watch) {
    await ctx.watch()
    console.log('[watch] build finished')
  } else {
    await ctx.rebuild()
    console.log('[build] build finished')
    await ctx.dispose()
  }
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})