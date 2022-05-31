#!/usr/bin/env node

import path from 'node:path';
import type { Argv } from 'yargs';
import yargs from 'yargs';

export const extension = path.extname(__filename).slice(1);
export const args = yargs
  .commandDir('commands', { extensions: [extension] })
  .demandCommand()
  .strictCommands()
  .env('SUBSTREAMS')
  .help()
  .parse();

type UnpackPromise<T> = T extends PromiseLike<infer U> ? U : T;

export type CommonArgs = UnpackPromise<typeof args>;
export type CommandArgs<T extends (argv: Argv) => Argv> = CommonArgs & UnpackPromise<ReturnType<T>['argv']>;
