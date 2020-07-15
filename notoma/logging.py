import logging

LOG_FMT = "%(asctime)s %(name)s [%(levelname)s]: %(message)s -- %(module)s.%(funcName)s %(filename)s:%(lineno)s"
LEVEL = logging.DEBUG
LOG_FNAME = ".notoma.log"

LOG_FILE_HANDLER = logging.FileHandler(LOG_FNAME, "w+")
LOG_NULL_HANDLER = logging.NullHandler()


def get_logger(level=LEVEL, handler=LOG_NULL_HANDLER, format=LOG_FMT):
    "Returns a customized logger. Defaults to logging INFO to NullHandler."
    handler.setFormatter(logging.Formatter(format))
    logger = logging.getLogger(__name__)
    logger.setLevel(LEVEL)
    logger.addHandler(handler)
    logger.info(f"New Notoma logger initialized with handler type: {type(handler)}")
    return logger


def toggle_debug(logger: logging.Logger, debug: bool = False) -> None:
    """
    Sets log level to INFO or DEBUG depending on a boolean flag in place,
    and returns None.
    """
    if debug:
        logger.info("Setting log level to DEBUG")
        logger.setLevel(logging.DEBUG)  # debug
    else:
        logger.info("Setting log level to INFO")
        logger.setLevel(logging.INFO)  # info
    return logger
