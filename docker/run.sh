#!/bin/bash

rm -f ${SSH_AUTH_SOCK} &&\

ssh-agent -a ${SSH_AUTH_SOCK} &&\

./app
