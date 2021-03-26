FROM php:7.1-apache-stretch

ENV PORT 80
EXPOSE 80

RUN docker-php-ext-install mysqli && docker-php-ext-enable mysqli

COPY . /var/www/html

RUN usermod -u 1000 www-data; \
    a2enmod rewrite; \
    chown -R www-data:www-data /var/www/html
