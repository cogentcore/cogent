git remote add upstream https://521github.com/goki/taskmanager.git
git remote -v
git fetch upstream
git checkout main
git stash
git rebase upstream/main
git stash pop