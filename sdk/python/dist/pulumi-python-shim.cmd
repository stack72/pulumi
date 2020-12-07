@echo off
REM Used to wrap python interpreters affected by https://github.com/golang/go/issues/42919
REM Expect first argument is the path to the interpreter to invoke. The rest of the args
REM are passed to the interpreter without modification

set "python_interpreter=%1"

SHIFT

set params=%1
:loop
shift
if [%1]==[] goto afterloop
set params=%params% %1
goto loop
:afterloop

%python_interpreter% %params%
