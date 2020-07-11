import logging

FMT = "%(asctime)s %(name)s [%(levelname)s]: %(message)s -- %(module)s.%(funcName)s %(filename)s:%(lineno)s"
LEVEL = logging.DEBUG
LOG_FNAME = ".notoma.log"
HANDLER = logging.FileHandler(LOG_FNAME, "w+")


def get_logger(level=LEVEL, handler=HANDLER, format=FMT):
    "Returns a customized logger. Defaults to logging INFO to `.notoma.log`."
    handler.setFormatter(logging.Formatter(format))
    logger = logging.getLogger(__name__)
    logger.setLevel(LEVEL)
    logger.addHandler(handler)
    return logger


def set_log_level_from_option(logger: logging.Logger, debug: bool = False) -> None:
    if debug:
        logger.info("Setting log level to DEBUG")
        logger.setLevel(logging.DEBUG)  # debug
    else:
        logger.info("Setting log level to INFO")
        logger.setLevel(logging.INFO)  # info
    return logger


logger = get_logger()
logger.info("Initializing logger for Notoma.")
