from setuptools import setup, find_packages

setup(
    name='snapshot',
    version='0.9.0',
    description='Process utilization (CUP, memory, io) monitoring',
    url='https://gadget.ncsl.nist.gov:8081/lizhong/snapshot',
    license='MIT',

    author='Lizhong Zhang',
    author_email='lizhong.zhang@nist.gov',

    pakcages='app',
    install_requires=['psutil', 'rfc3339', 'simplejson'],
)