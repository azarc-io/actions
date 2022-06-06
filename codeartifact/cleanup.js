const core = require('@actions/core');
const exec = require('@actions/exec');
const io = require('@actions/io');

async function cleanup() {
  console.log('cleaning up .npmrc')
  await io.rmRF('.npmrc');
  await io.rmRF('gradle.properties');
}

module.exports = cleanup;

/* istanbul ignore next */
if (require.main === module) {
  cleanup();
}
