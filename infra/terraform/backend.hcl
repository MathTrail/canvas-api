# Where the state lives: the bucket infra/bootstrap.sh creates before the first
# init, because a backend cannot create the place it keeps its state in.

bucket = "mathtrail-canvas-prod-tfstate"
prefix = "canvas-api"
